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
	"strings"
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
	return &TokenDirectory{
		options:        opts,
		client:         client,
		tokenListCache: map[string]TokenList{},
	}
}

type Options struct {
	// HTTPClient is the HTTP client to use for fetching the token directory.
	//
	// Default is http.DefaultClient.
	HTTPClient *http.Client

	// ChainIDs is a list of chain IDs to fetch, acting as a filter on top of the index.
	// If not provided, all chain IDs will be fetched.
	//
	// Default is nil, which means all chain IDs will be fetched.
	ChainIDs []uint64

	// SkipExternalTokenLists is a flag to skip fetching external token lists.
	// The external token lists are external lists which are imported into
	// the token directory.
	//
	// Default is false, meaning external token lists will be fetched.
	SkipExternalTokenLists bool

	// IncludeDeprecated is a flag to include deprecated token lists.
	// If not provided, deprecated token lists will be skipped.
	//
	// Default is false, meaning deprecated token lists will be skipped.
	IncludeDeprecated bool

	// TokenListURLs is a list of token list URLs to fetch, acting
	// as a filter on top of the index to only ever fetch these
	// urls. If not provided, all token list URLs will be fetched.
	//
	// Default is nil, which means all token list URLs will be fetched.
	TokenListURLs []string

	// OnlyERC20 is a flag to only include ERC20 token lists.
	//
	// Default is false, meaning all token standards will be included.
	OnlyERC20 bool

	// NoCache is a flag to disable the local token list cache.
	// The cache works by checking the content hash of the Index
	// with the content hash of the TokenList. If the content hash
	// has changed, the TokenList will be refetched.
	//
	// Default is false, therefore the cache is enabled.
	NoCache bool
}

const tokenDirectoryBaseSourceURL = "https://raw.githubusercontent.com/0xsequence/token-directory/master/index"

type TokenDirectory struct {
	options Options
	client  *http.Client

	index          TokenDirectoryIndex
	indexFetchedAt time.Time

	tokenListCache map[string]TokenList

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
	index, err := d.fetchIndex(ctx, optFilter...)
	if err != nil {
		return nil, err
	}

	// Create a deep copy of the index
	result := TokenDirectoryIndex{}
	for chainID, entries := range index {
		entriesCopy := make([]TokenDirectoryIndexEntry, len(entries))
		copy(entriesCopy, entries)
		result[chainID] = entriesCopy
	}

	return result, nil
}

func (d *TokenDirectory) fetchIndex(ctx context.Context, optFilter ...IndexFilter) (TokenDirectoryIndex, error) {
	var filter *IndexFilter
	if len(optFilter) > 0 {
		filter = &optFilter[0]
	}

	// we memoize the index for 30 seconds to refrain from fetching from
	// the remote source too often.
	d.mu.Lock()
	indexFetchedAt := d.indexFetchedAt
	if time.Since(indexFetchedAt) < 30*time.Second {
		tdIndex := filteredIndex(d.index, filter)
		d.mu.Unlock()
		return tdIndex, nil
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
			if name != "_external" && d.options.OnlyERC20 && file != "erc20.json" {
				continue
			}

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
	d.index = tdIndex
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
	index, err := d.fetchIndex(ctx)
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
	// index, err := d.fetchIndex(ctx, IndexFilter{ChainIDs: []uint64{chainID}})
	// if err != nil {
	// 	return nil, err
	// }
	// out, err := d.FetchTokenLists(ctx, index)
	// if err != nil {
	// 	return nil, err
	// }
	// tokenLists, ok := out[chainID]
	// if !ok {
	// 	return nil, fmt.Errorf("tokendirectory: no token lists found")
	// }
	// return tokenLists, nil
}

func (d *TokenDirectory) FetchExternalTokenLists(ctx context.Context) ([]TokenList, error) {
	index, err := d.fetchIndex(ctx)
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
	// index, err := d.fetchIndex(ctx, IndexFilter{External: true})
	// if err != nil {
	// 	return nil, err
	// }
	// out, err := d.FetchTokenLists(ctx, index)
	// if err != nil {
	// 	return nil, err
	// }
	// tokenLists, ok := out[0]
	// if !ok {
	// 	return nil, fmt.Errorf("tokendirectory: no token lists found")
	// }
	// return tokenLists, nil
}

func (d *TokenDirectory) FetchTokenLists(ctx context.Context, index TokenDirectoryIndex) (map[uint64][]TokenList, error) {
	tokenLists := map[uint64][]TokenList{}
	for chainID, entries := range index {
		tokenLists[chainID] = []TokenList{}
		for _, entry := range entries {
			tokenList, err := d.FetchTokenList(ctx, entry.TokenListURL)
			if err != nil {
				return nil, err
			}
			tokenLists[chainID] = append(tokenLists[chainID], tokenList)
		}
	}

	return tokenLists, nil
}

func (d *TokenDirectory) FetchTokenContractInfo(ctx context.Context, index TokenDirectoryIndex) (map[uint64][]ContractInfo, error) {
	tokenListMap, err := d.FetchTokenLists(ctx, index)
	if err != nil {
		return nil, err
	}

	contractInfoMap := map[uint64][]ContractInfo{}

	// first include external token sources, as other lists will override to take
	// precedence per chainID
	externalList, ok := tokenListMap[0]
	if ok {
		for _, tokenList := range externalList {
			contractInfoList := tokenList.Tokens
			for _, ci := range contractInfoList {
				chainID := ci.ChainID
				if chainID == 0 {
					return nil, fmt.Errorf("tokendirectory: token list contains token with chainID 0: %s", tokenList.TokenListURL)
				}
				if _, ok := contractInfoMap[chainID]; !ok {
					contractInfoMap[chainID] = []ContractInfo{}
				}
				contractInfoMap[chainID] = append(contractInfoMap[chainID], ci)
			}
		}
	}

	// then include chain specific token lists, which will override external list
	for tokenListChainID, tokenLists := range tokenListMap {
		if tokenListChainID == 0 {
			continue
		}
		for _, tokenList := range tokenLists {
			contractInfoList := tokenList.Tokens
			for _, ci := range contractInfoList {
				if ci.ChainID == 0 {
					return nil, fmt.Errorf("tokendirectory: token list contains token with chainID 0: %s", tokenList.TokenListURL)
				}
			}
			if _, ok := contractInfoMap[tokenListChainID]; !ok {
				contractInfoMap[tokenListChainID] = contractInfoList
			} else {
				contractInfoMap[tokenListChainID] = append(contractInfoMap[tokenListChainID], contractInfoList...)
			}
		}
	}

	// sort and deduplicate contract info per chainID
	for chainID, contractInfos := range contractInfoMap {
		uniqueMap := map[string]ContractInfo{}
		for _, ci := range contractInfos {
			key := fmt.Sprintf("%d-%s", ci.ChainID, ci.Address)
			if ci.Address == "0xeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee" {
				// we skip the 0xee..ee entry, as we assume there is a 0x00..00 entry
				// and prefer to avoid duplicates for the native token
				continue
			}
			if ci.Address == "0x0000000000000000000000000000000000000000" {
				ci.Extensions.Featured = true
				ci.Extensions.FeatureIndex = -1000000 // ensure native tokens are always at the top
			}
			if ci.Extensions.FeatureIndex == 0 {
				ci.Extensions.FeatureIndex = 1000000 // ensure non-featured tokens are at the bottom
			}
			uniqueMap[key] = ci // last one wins
		}
		uniqueList := []ContractInfo{}
		for _, ci := range uniqueMap {
			uniqueList = append(uniqueList, ci)
		}
		sort.Slice(uniqueList, func(i, j int) bool {
			fi := uniqueList[i].Extensions.FeatureIndex
			fj := uniqueList[j].Extensions.FeatureIndex
			if fi != fj {
				return fi < fj // lower FeatureIndex first (ie. think like rank position: 1,2,3,etc.)
			}
			return uniqueList[i].Name < uniqueList[j].Name // then alpha by Name
		})
		contractInfoMap[chainID] = uniqueList
	}

	return contractInfoMap, nil
}

func (d *TokenDirectory) GetContentHashForTokenList(ctx context.Context, tokenListURL string) (string, bool, error) {
	index, err := d.fetchIndex(ctx)
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

func (d *TokenDirectory) GetChainTokenListURLs(ctx context.Context, chainID uint64) ([]string, []string, error) {
	index, err := d.fetchIndex(ctx)
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
	index, err := d.fetchIndex(ctx)
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
	if d.UseCache() {
		d.mu.Lock()
		tokenList, ok := d.tokenListCache[tokenListURL]
		d.mu.Unlock()

		if ok && tokenList.ContentHash != "" {
			indexedContentHash, ok, err := d.GetContentHashForTokenList(ctx, tokenListURL)
			if err != nil {
				return TokenList{}, fmt.Errorf("tokendirectory: failed to get content hash for token list %s: %w", tokenListURL, err)
			}
			if ok && tokenList.ContentHash == indexedContentHash {
				return tokenList, nil
			}
		}
	}

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
	index, _ := d.fetchIndex(ctx)
	for _, entries := range index {
		for _, entry := range entries {
			if entry.TokenListURL == tokenListURL {
				deprecated = entry.Deprecated
				break
			}
		}
	}
	tokenList.Deprecated = deprecated

	// When d.Options is configured with ChainIDs, then we will filter the token lists
	// to only include the tokens that match the chainIDs. This is handy for
	// external token lists to avoid copying their data for chains which are not
	// of interest.
	if d.options.ChainIDs != nil {
		chainIDs := d.options.ChainIDs
		if tokenList.ChainID > 0 && !slices.Contains(chainIDs, tokenList.ChainID) {
			// token list is not for a chain of interest, return empty set
			tokenList.Tokens = []ContractInfo{}
		} else if tokenList.ChainID == 0 {
			// token list is an external token list, filter the tokens to only include
			// the tokens that match the chainIDs
			tokens := []ContractInfo{}
			for _, token := range tokenList.Tokens {
				if !slices.Contains(chainIDs, token.ChainID) {
					continue
				}
				tokens = append(tokens, token)
			}
			tokenList.Tokens = tokens
		} else {
			// all is good, no need to filter
		}
	}

	// normalize/downcase all contract addresses in the token list
	for i, token := range tokenList.Tokens {
		tokenList.Tokens[i].Address = strings.ToLower(token.Address)
		tokenList.Tokens[i].Name = strings.TrimSpace(token.Name)
		tokenList.Tokens[i].Symbol = strings.TrimSpace(token.Symbol)
	}

	// Cache the token list if caching is enabled. Note: this will be evicted
	// very quickly if the index is updated.
	if d.UseCache() {
		d.mu.Lock()
		d.tokenListCache[tokenListURL] = tokenList
		d.mu.Unlock()
	}

	return tokenList, nil
}

func (d *TokenDirectory) UseCache() bool {
	return !d.options.NoCache
}

// DiffIndex returns the difference between two token directory indexes, focusing on
// what's new or changed in index2 (the newer version) compared to index1. Think
// of index1 like the first version of the index, and index2 like the second version.
//
// The diff logic creates a new index containing:
// 1. Entries that exist in index2 but not in index1 (new entries)
// 2. Entries that exist in both but have different content hashes (changed entries)
// In all cases, the index2 version of the entry is used in the output.
func DiffIndex(index1, index2 TokenDirectoryIndex) TokenDirectoryIndex {
	if index1 == nil {
		return index2
	}
	if index2 == nil {
		return TokenDirectoryIndex{}
	}
	out := TokenDirectoryIndex{}

	// Check for entries in index2 that are new or different from index1
	for chainID2, entries2 := range index2 {
		for _, entry2 := range entries2 {
			found := false
			if entries1, exists := index1[chainID2]; exists {
				for _, entry1 := range entries1 {
					if entry2.TokenListURL == entry1.TokenListURL {
						found = true
						// If content hash is different, add to diff (using entry2)
						if entry2.ContentHash != entry1.ContentHash {
							if _, ok := out[chainID2]; !ok {
								out[chainID2] = []TokenDirectoryIndexEntry{}
							}
							out[chainID2] = append(out[chainID2], entry2)
						}
						break
					}
				}
			}
			// If entry doesn't exist in index1, add to diff
			if !found {
				if _, ok := out[chainID2]; !ok {
					out[chainID2] = []TokenDirectoryIndexEntry{}
				}
				out[chainID2] = append(out[chainID2], entry2)
			}
		}
	}

	// Sort entries for consistency
	for chainID, entries := range out {
		sort.Slice(entries, func(i, j int) bool {
			return entries[i].Filename < entries[j].Filename
		})
		out[chainID] = entries
	}

	return out
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
	} else {
		for chainID, entries := range index {
			if chainID != 0 {
				out[chainID] = entries
			}
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
