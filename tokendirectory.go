package tokendirectory

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
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

// Note: these are vars (not consts) only so that tests can point them at
// httptest servers. Tests that mutate them must not run in parallel.
var (
	// tokenDirectoryBaseSourceURL is the primary source for the token directory
	// index and token lists, served from the GitHub repository.
	tokenDirectoryBaseSourceURL = "https://raw.githubusercontent.com/0xsequence/token-directory/master/index"

	// tokenDirectoryFallbackSourceURL is the fallback source for the token
	// directory index and token lists, served from a public GCS bucket mirror.
	// It is used when the primary source is unavailable (e.g. rate limited or
	// returning errors).
	tokenDirectoryFallbackSourceURL = "https://storage.googleapis.com/token-directory-index/index"
)

// sourceAttemptTimeout bounds how long a single source (primary or fallback)
// can take before we move on to the next, so a stalled primary still leaves
// room for the mirror. It is a var only so tests can shorten it.
var sourceAttemptTimeout = 10 * time.Second

// ErrSourceTimeout is reported (wrapped) when a single source exceeds
// sourceAttemptTimeout while the caller's context is still alive. It lets
// callers distinguish "the source was slow" (safe to retry) from the
// caller's own context deadline expiring.
var ErrSourceTimeout = errors.New("source timed out")

type TokenDirectory struct {
	options Options
	client  *http.Client

	index          TokenDirectoryIndex
	indexFetchedAt time.Time
	preferFallback bool

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

type tokenDirectoryIndexFile struct {
	Index map[string]struct {
		ChainID    uint64            `json:"chainId"`
		Deprecated bool              `json:"deprecated"`
		TokenLists map[string]string `json:"tokenLists"`
	} `json:"index"`
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

	// Fetch the index from the primary (GitHub) source, falling back to the
	// GCS mirror if the primary is unavailable.
	var indexFile tokenDirectoryIndexFile
	validateIndex := func(buf []byte) error {
		var candidate tokenDirectoryIndexFile
		if err := json.Unmarshal(buf, &candidate); err != nil {
			return fmt.Errorf("unmarshalling index.json: %w", err)
		}
		if candidate.Index == nil {
			return fmt.Errorf("index.json is missing index")
		}
		indexFile = candidate
		return nil
	}

	_, err := d.fetchManagedURLs(
		ctx,
		TokenDirectoryIndexURL(),
		TokenDirectoryFallbackIndexURL(),
		true,
		validateIndex,
	)
	if err != nil {
		return nil, fmt.Errorf("tokendirectory: fetching index.json: %w", err)
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
			tokenList, err := d.fetchTokenList(ctx, entry.TokenListURL, entry.ContentHash)
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
			tokenList, err := d.fetchTokenList(ctx, entry.TokenListURL, entry.ContentHash)
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
			tokenList, err := d.fetchTokenList(ctx, entry.TokenListURL, entry.ContentHash)
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
	var expectedContentHash string
	if fallbackURLFor(tokenListURL) != tokenListURL {
		if hash, ok, err := d.GetContentHashForTokenList(ctx, tokenListURL); err == nil && ok {
			expectedContentHash = hash
		}
	}
	return d.fetchTokenList(ctx, tokenListURL, expectedContentHash)
}

func (d *TokenDirectory) fetchTokenList(ctx context.Context, tokenListURL string, expectedContentHash string) (TokenList, error) {
	if d.UseCache() {
		d.mu.Lock()
		tokenList, ok := d.tokenListCache[tokenListURL]
		d.mu.Unlock()

		if ok && tokenList.ContentHash != "" {
			indexedContentHash := expectedContentHash
			indexedContentHashFound := indexedContentHash != ""
			if !indexedContentHashFound {
				var err error
				indexedContentHash, indexedContentHashFound, err = d.GetContentHashForTokenList(ctx, tokenListURL)
				if err != nil {
					return TokenList{}, fmt.Errorf("tokendirectory: failed to get content hash for token list %s: %w", tokenListURL, err)
				}
			}
			if indexedContentHashFound && tokenList.ContentHash == indexedContentHash {
				return tokenList, nil
			}
		}
	}

	var tokenList TokenList
	var contentHash string
	validateTokenList := func(buf []byte) error {
		var candidate TokenList
		if err := json.Unmarshal(buf, &candidate); err != nil {
			return fmt.Errorf("unmarshalling token list: %w", err)
		}
		candidateHash := sha256Hash(buf)
		if expectedContentHash != "" && candidateHash != expectedContentHash {
			return fmt.Errorf("content hash mismatch: expected %s, got %s", expectedContentHash, candidateHash)
		}
		tokenList = candidate
		contentHash = candidateHash
		return nil
	}

	var err error
	if fallback := fallbackURLFor(tokenListURL); fallback != tokenListURL {
		_, err = d.fetchManagedURLs(ctx, tokenListURL, fallback, false, validateTokenList)
	} else {
		_, err = d.fetchFromSources(ctx, validateTokenList, fetchSource{url: tokenListURL})
	}
	if err != nil {
		return TokenList{}, fmt.Errorf("tokendirectory: failed to fetch token list %s: %w", tokenListURL, err)
	}

	tokenList.TokenListURL = tokenListURL
	tokenList.ContentHash = contentHash

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

func TokenDirectoryFallbackIndexURL() string {
	return fmt.Sprintf("%s/index.json", tokenDirectoryFallbackSourceURL)
}

func TokenDirectoryFallbackTokenListURL(group string, file string) string {
	return fmt.Sprintf("%s/%s/%s", tokenDirectoryFallbackSourceURL, group, file)
}

// fallbackURLFor returns the fallback (GCS mirror) URL for the given primary
// (GitHub) URL. URLs not served from the primary source are returned unchanged.
func fallbackURLFor(url string) string {
	if rest, ok := strings.CutPrefix(url, tokenDirectoryBaseSourceURL); ok {
		return tokenDirectoryFallbackSourceURL + rest
	}
	return url
}

type fetchSource struct {
	url       string
	isPrimary bool
}

type responseValidator func([]byte) error

// fetchManagedURLs fetches a primary/fallback pair. Index refreshes probe the
// primary so it can recover; token-list requests prefer the fallback after a
// primary failure until the next index refresh.
func (d *TokenDirectory) fetchManagedURLs(
	ctx context.Context,
	primaryURL string,
	fallbackURL string,
	probePrimary bool,
	validate responseValidator,
) ([]byte, error) {
	d.mu.Lock()
	preferFallback := d.preferFallback
	d.mu.Unlock()

	sources := []fetchSource{
		{url: primaryURL, isPrimary: true},
		{url: fallbackURL},
	}
	if preferFallback && !probePrimary {
		slices.Reverse(sources)
	}
	return d.fetchFromSources(ctx, validate, sources...)
}

// fetchFromURLs fetches the given URLs in order and returns the body of the
// first URL that responds with 200 OK. This is used to transparently fall
// back to the GCS mirror when the primary GitHub source is unavailable
// (e.g. rate limited, 5xx, stalled, or network errors). An error is returned
// only if all URLs fail, joining the individual failures so no error is lost.
func (d *TokenDirectory) fetchFromURLs(ctx context.Context, urls ...string) ([]byte, error) {
	sources := make([]fetchSource, len(urls))
	for i, url := range urls {
		sources[i] = fetchSource{url: url}
	}
	return d.fetchFromSources(ctx, nil, sources...)
}

func (d *TokenDirectory) fetchFromSources(ctx context.Context, validate responseValidator, sources ...fetchSource) ([]byte, error) {
	if len(sources) == 0 {
		return nil, fmt.Errorf("no urls provided")
	}
	var errs []error
	for _, source := range sources {
		// bail out early if the caller canceled or timed out, keeping the
		// errors collected so far
		if err := ctx.Err(); err != nil {
			return nil, errors.Join(append(errs, err)...)
		}

		attemptCtx := ctx
		cancel := func() {}
		if len(sources) > 1 {
			// Give failover sources their own budgets. A single arbitrary URL
			// continues to honor only the caller context and configured client.
			attemptCtx, cancel = context.WithTimeout(ctx, sourceAttemptTimeout)
		}
		buf, err := d.fetchOnce(attemptCtx, source.url)
		if err == nil && validate != nil {
			if validationErr := validate(buf); validationErr != nil {
				err = fmt.Errorf("validating response: %w", validationErr)
			}
		}
		cancel()
		if err == nil {
			if source.isPrimary {
				d.mu.Lock()
				d.preferFallback = false
				d.mu.Unlock()
			}
			return buf, nil
		}
		if source.isPrimary && ctx.Err() == nil {
			d.mu.Lock()
			d.preferFallback = true
			d.mu.Unlock()
		}
		// if the attempt timed out but the caller's context is still alive,
		// surface a source timeout rather than the attempt's deadline, so
		// callers don't mistake it for their own budget being spent
		if ctx.Err() == nil && errors.Is(err, context.DeadlineExceeded) {
			err = fmt.Errorf("%w: %v", ErrSourceTimeout, err)
		}
		errs = append(errs, fmt.Errorf("fetching %s: %w", source.url, err))
	}
	return nil, errors.Join(errs...)
}

// fetchOnce fetches a single URL and returns its body if it responds with
// 200 OK. The body is fully read before returning, so the caller may cancel
// the context afterwards.
func (d *TokenDirectory) fetchOnce(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	res, err := d.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		// drain the body so the connection can be reused
		_, _ = io.Copy(io.Discard, res.Body)
		return nil, fmt.Errorf("status %s", res.Status)
	}
	buf, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("reading body: %w", err)
	}
	return buf, nil
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
