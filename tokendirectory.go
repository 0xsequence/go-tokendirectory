package tokendirectory

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"slices"
	"sort"
	"sync"
	"time"
)

const tokenDirectoryBaseSourceURL = "https://raw.githubusercontent.com/0xsequence/token-directory/master/index"

type Provider interface {
	FetchTokenList(ctx context.Context, url string) (TokenList, error)
}

type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

func TokenDirectoryIndexURL() string {
	return fmt.Sprintf("%s/index.json", tokenDirectoryBaseSourceURL)
}

func TokenDirectoryTokenListURL(group string, file string) string {
	return fmt.Sprintf("%s/%s/%s", tokenDirectoryBaseSourceURL, group, file)
}

// func NewTokenListProvider(sourceURLs []string) (Provider, error) {
// 	if len(sourceURLs) == 0 {
// 		return nil, fmt.Errorf("no source URLs provided")
// 	}
// 	return &tokenListProvider{sourceURLs: sourceURLs}, nil
// }

// type tokenListProvider struct {
// 	client     *http.Client // TODO
// 	sourceURLs []string
// }

// var _ Provider = &tokenListProvider{}

// func (p *tokenListProvider) FetchTokenList(ctx context.Context, url string) (*TokenList, error) {
// 	return nil, nil
// }

//--

type TokenDirectory struct {
	options Options
	client  *http.Client

	index          TokenDirectoryIndex
	indexFetchedAt time.Time

	mu sync.Mutex
}

// TODO: we can add a filter by chainID ..
// TODO: we can add filter to include external or not ..

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

// TODO: we can limit to only some number of chainIds ..
// or pass nil for all.. or just have it in Options.ChainIDs []uint64 ..

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

// TODO: .. so lets track the last time we watched the index .. and lets
// set option that we will only re-fetch every X time..

type IndexFilter struct {
	ChainIDs []uint64
	External bool
}

// TODO: add optFilter ...IndexFilter
func (d *TokenDirectory) FetchIndex(ctx context.Context, optFilter ...IndexFilter) (TokenDirectoryIndex, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	filter := IndexFilter{}
	if len(optFilter) > 0 {
		filter = optFilter[0]
	}
	_ = filter

	// TODO .....

	if time.Since(d.indexFetchedAt) < 30*time.Second {
		fmt.Println("serving here")
		return d.index, nil
	}
	fmt.Println("fetching index")

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
			continue // skipping for now + will add option later
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

	d.index = tdIndex
	d.indexFetchedAt = time.Now()

	return tdIndex, nil
}

type TokenDirectoryIndex map[uint64][]TokenDirectoryIndexEntry

type TokenDirectoryIndexEntry struct {
	ChainID      uint64
	Deprecated   bool
	Filename     string
	ContentHash  string
	TokenListURL string
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

func (d *TokenDirectory) GetChainTokenListURLs(ctx context.Context, chainID uint64) ([]string, error) {
	index, err := d.FetchIndex(ctx)
	if err != nil {
		return nil, err
	}
	urls := []string{}
	for _, entry := range index[chainID] {
		urls = append(urls, entry.TokenListURL)
	}
	return urls, nil
}

// TODO: maybe we pass optional, "lastContentHash", so if this is it, we skip ..
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

// TODO: keep track of sourceURL and their hash .. we can keep track of it in memory
// .. we can also say, forceFetch ..

func sha256Hash(data []byte) string {
	h := sha256.New()
	h.Write(data)
	return hex.EncodeToString(h.Sum(nil))
}
