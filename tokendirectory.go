package tokendirectory

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"net/http"
	"slices"
	"sort"
	"sync"
	"time"
)

// NewTokenDirectory creates a new TokenDirectory instance which provides
// access to the token directory index and token lists. By default with
// no custom options, it will fetch all chains, all external token lists,
// but will skip deprecated chain token lists.
func NewTokenDirectory(options ...Options) *TokenDirectory {
	opts := Options{}
	if len(options) > 0 {
		opts = options[0]
	}
	client := http.DefaultClient
	if opts.HTTPClient != nil {
		client = opts.HTTPClient
	}
	return &TokenDirectory{options: opts, client: client}
}

type Options struct {
	// HTTPClient is the HTTP client to use for fetching the token directory.
	// The default is http.DefaultClient.
	HTTPClient *http.Client

	// ChainIDs is a list of chain IDs to fetch, acting as a filter on top of the index.
	// If not provided, all chain IDs will be fetched.
	ChainIDs []uint64

	// SkipExternalTokenLists is a flag to skip fetching external token lists.
	// The external token lists are external lists which are imported into
	// the token directory.
	SkipExternalTokenLists bool

	// IncludeDeprecated is a flag to include deprecated token lists.
	// If not provided, deprecated token lists will be skipped.
	IncludeDeprecated bool

	// TokenListURLs is a list of token list URLs to fetch, acting
	// as a filter on top of the index to only ever fetch these
	// urls. If not provided, all token list URLs will be fetched.
	TokenListURLs []string
}

const tokenDirectoryBaseSourceURL = "https://raw.githubusercontent.com/0xsequence/token-directory/master/index"

type TokenDirectory struct {
	options Options
	client  *http.Client

	index          TokenDirectoryIndex
	indexFetchedAt time.Time

	mu sync.Mutex
}

type IndexFilter struct {
	// All flag will return everything, aka no filtering.
	All bool

	// ChainIDs flag will return just the specific chains.
	ChainIDs []uint64

	// External flag will return just the external token lists
	// aka, chainID 0.
	External bool

	// Deprecated flag will return just the deprecated token lists.
	Deprecated bool
}

func (d *TokenDirectory) FetchIndex(ctx context.Context, optFilter ...IndexFilter) (TokenDirectoryIndex, error) {
	var filter *IndexFilter
	if len(optFilter) > 0 {
		filter = &optFilter[0]
	}

	// we memoize the index for 30 seconds to refrain from fetching from
	// the remote source too often.
	d.mu.Lock()
	indexFetchedAt := d.indexFetchedAt
	if time.Since(indexFetchedAt) < 30*time.Second {
		index := maps.Clone(d.index)
		d.mu.Unlock()
		return filteredIndex(index, filter), nil
	}
	d.mu.Unlock()

	req, err := http.NewRequest("GET", TokenDirectoryIndexURL(), nil)
	if err != nil {
		return nil, fmt.Errorf("tokendirectory: creating request: %w", err)
	}
	res, err := d.client.Do(req.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("tokendirectory: fetching index.json: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("tokendirectory: fetching index.json: %s", res.Status)
	}

	buf, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("tokendirectory: reading index.jsonbody: %w", err)
	}

	var indexFile struct {
		Index map[string]struct {
			ChainID    uint64            `json:"chainId"`
			Deprecated bool              `json:"deprecated"`
			TokenLists map[string]string `json:"tokenLists"`
		} `json:"index"`
	}

	if err := json.Unmarshal(buf, &indexFile); err != nil {
		return nil, fmt.Errorf("tokendirectory: unmarshalling index.json: %w", err)
	}

	tdIndex := TokenDirectoryIndex{}

	for name, group := range indexFile.Index {
		if d.options.SkipExternalTokenLists && name == "_external" {
			continue
		}

		chainID := group.ChainID
		deprecated := group.Deprecated
		tokenLists := group.TokenLists

		if name != "_external" && chainID == 0 {
			// extra sanity check, even though the index should never produce this
			// TODO: we could log the error too
			// err := fmt.Errorf("tokendirectory: index chainId is 0 for %s", name)
			continue
		}

		if chainID > 0 && len(d.options.ChainIDs) > 0 && !slices.Contains(d.options.ChainIDs, chainID) {
			continue
		}

		if !d.options.IncludeDeprecated && deprecated {
			continue
		}

		for file, hash := range tokenLists {
			tokenListURL := TokenDirectoryTokenListURL(name, file)
			if len(d.options.TokenListURLs) > 0 && !slices.Contains(d.options.TokenListURLs, tokenListURL) {
				continue
			}

			if _, ok := tdIndex[chainID]; !ok {
				tdIndex[chainID] = []TokenDirectoryIndexEntry{}
			}

			tdIndex[chainID] = append(tdIndex[chainID], TokenDirectoryIndexEntry{
				ChainID:      chainID,
				Deprecated:   deprecated,
				Filename:     file,
				ContentHash:  hash,
				TokenListURL: tokenListURL,
			})
		}

		sort.Slice(tdIndex[chainID], func(i, j int) bool {
			return tdIndex[chainID][i].Filename < tdIndex[chainID][j].Filename
		})
	}

	d.mu.Lock()
	d.index = maps.Clone(tdIndex)
	d.indexFetchedAt = time.Now()
	d.mu.Unlock()

	return filteredIndex(tdIndex, filter), nil
}

type TokenDirectoryIndex map[uint64][]TokenDirectoryIndexEntry

type TokenDirectoryIndexEntry struct {
	ChainID      uint64
	Deprecated   bool
	Filename     string
	ContentHash  string
	TokenListURL string
}

func (d *TokenDirectory) FetchChainTokenLists(ctx context.Context, chainID uint64) ([]TokenList, error) {
	index, err := d.FetchIndex(ctx)
	if err != nil {
		return nil, err
	}

	tokenLists := []TokenList{}

	for indexChainID, indexEntries := range index {
		if indexChainID != chainID {
			continue
		}

		for _, entry := range indexEntries {
			tokenList, err := d.FetchTokenList(ctx, entry.TokenListURL)
			if err != nil {
				return nil, err
			}
			tokenLists = append(tokenLists, tokenList)
		}
	}

	return tokenLists, nil
}

func (d *TokenDirectory) FetchExternalTokenLists(ctx context.Context) ([]TokenList, error) {
	index, err := d.FetchIndex(ctx)
	if err != nil {
		return nil, err
	}

	tokenLists := []TokenList{}

	for indexChainID, indexEntries := range index {
		if indexChainID != 0 {
			continue
		}

		for _, entry := range indexEntries {
			tokenList, err := d.FetchTokenList(ctx, entry.TokenListURL)
			if err != nil {
				return nil, err
			}
			tokenLists = append(tokenLists, tokenList)
		}
	}

	return tokenLists, nil
}

func (d *TokenDirectory) FetchTokenLists(ctx context.Context, index TokenDirectoryIndex) (map[uint64][]TokenList, error) {
	tokenLists := map[uint64][]TokenList{}
	for chainID, entries := range index {
		tokenLists[chainID] = []TokenList{}
		for _, entry := range entries {
			// TODO: since we have our index with a content hash .. and we now fetch again
			// it should be nice to inform if its changed since last index or not ..

			tokenList, err := d.FetchTokenList(ctx, entry.TokenListURL)
			if err != nil {
				return nil, err
			}
			tokenLists[chainID] = append(tokenLists[chainID], tokenList)
		}
	}
	return tokenLists, nil
}

func (d *TokenDirectory) GetContentHashForTokenList(ctx context.Context, tokenListURL string) (string, bool, error) {
	index, err := d.FetchIndex(ctx)
	if err != nil {
		return "", false, err
	}
	for _, entries := range index {
		for _, entry := range entries {
			if entry.TokenListURL == tokenListURL {
				return entry.ContentHash, true, nil
			}
		}
	}
	return "", false, nil
}

func (d *TokenDirectory) IsTokenListContentStale(ctx context.Context, tokenListURL string) (bool, error) {
	index, err := d.FetchIndex(ctx)
	if err != nil {
		return false, err
	}

	// TODO ..... so we kinda need to store the "prev" index .. so we can check if its changed..?
	// this endpoint will be kinda tricky...

	// for _, entries := range index {
	// 	for _, entry := range entries {
	_ = index
	return false, nil
}

func (d *TokenDirectory) GetChainTokenListURLs(ctx context.Context, chainID uint64) ([]string, []string, error) {
	index, err := d.FetchIndex(ctx)
	if err != nil {
		return nil, nil, err
	}
	urls := []string{}
	hashes := []string{}
	for _, entry := range index[chainID] {
		urls = append(urls, entry.TokenListURL)
		hashes = append(hashes, entry.ContentHash)
	}
	return urls, hashes, nil
}

func (d *TokenDirectory) GetExternalTokenListURLs(ctx context.Context) ([]string, []string, error) {
	index, err := d.FetchIndex(ctx)
	if err != nil {
		return nil, nil, err
	}
	urls := []string{}
	hashes := []string{}
	for _, entry := range index[0] {
		urls = append(urls, entry.TokenListURL)
		hashes = append(hashes, entry.ContentHash)
	}
	return urls, hashes, nil
}

func (d *TokenDirectory) FetchTokenList(ctx context.Context, tokenListURL string) (TokenList, error) {
	req, err := http.NewRequest("GET", tokenListURL, nil)
	if err != nil {
		return TokenList{}, fmt.Errorf("tokendirectory: failed to create request: %w", err)
	}
	res, err := d.client.Do(req.WithContext(ctx))
	if err != nil {
		return TokenList{}, fmt.Errorf("tokendirectory: failed to fetch token list %s: %w", tokenListURL, err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return TokenList{}, fmt.Errorf("tokendirectory: failed to fetch token list %s", tokenListURL)
	}

	buf, err := io.ReadAll(res.Body)
	if err != nil {
		return TokenList{}, fmt.Errorf("tokendirectory: failed to read token list %s: %w", tokenListURL, err)
	}

	var tokenList TokenList
	if err := json.Unmarshal(buf, &tokenList); err != nil {
		return TokenList{}, fmt.Errorf("tokendirectory: failed to unmarshal token list %s: %w", tokenListURL, err)
	}

	tokenList.TokenListURL = tokenListURL
	tokenList.ContentHash = sha256Hash(buf)

	var deprecated bool
	index, _ := d.FetchIndex(ctx)
	for _, entries := range index {
		for _, entry := range entries {
			if entry.TokenListURL == tokenListURL {
				deprecated = entry.Deprecated
				break
			}
		}
	}
	tokenList.Deprecated = deprecated

	return tokenList, nil
}

func TokenDirectoryIndexURL() string {
	return fmt.Sprintf("%s/index.json", tokenDirectoryBaseSourceURL)
}

func TokenDirectoryTokenListURL(group string, file string) string {
	return fmt.Sprintf("%s/%s/%s", tokenDirectoryBaseSourceURL, group, file)
}

func filteredIndex(index TokenDirectoryIndex, filter *IndexFilter) TokenDirectoryIndex {
	if filter == nil || filter.All {
		return index
	}

	out := TokenDirectoryIndex{}

	if len(filter.ChainIDs) > 0 {
		for _, chainID := range filter.ChainIDs {
			out[chainID] = index[chainID]
		}
	}
	if filter.External {
		out[0] = index[0]
	}
	if filter.Deprecated {
		for chainID, entries := range index {
			deprecated := false
			for _, entry := range entries {
				if entry.Deprecated {
					deprecated = true
					break
				}
			}
			if deprecated {
				out[chainID] = entries
			}
		}
	}
	return out
}

func sha256Hash(data []byte) string {
	h := sha256.New()
	h.Write(data)
	return hex.EncodeToString(h.Sum(nil))
}
