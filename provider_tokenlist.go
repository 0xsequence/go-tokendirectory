package tokendirectory

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

func NewTokenListProvider(sources []map[uint64]map[SourceType]string, optHttpClient ...*http.Client) Provider {
	var source map[uint64]map[SourceType]string
	if len(sources) == 0 {
		source = MergeSources(SequenceGithubSources, UniswapSources, SushiSources, CoinGeckoSources, PancakeSources)
	} else {
		source = MergeSources(sources...)
	}

	httpClient := http.DefaultClient
	if len(optHttpClient) > 0 {
		httpClient = optHttpClient[0]
	}

	return tokenListProvider{
		id:      "tokenlist-directory",
		client:  httpClient,
		sources: source,
	}

}

type tokenListProvider struct {
	id      string
	client  *http.Client
	sources map[uint64]map[SourceType]string
}

func (p tokenListProvider) GetID() string {
	return p.id

}

func (p tokenListProvider) GetConfig(ctx context.Context) ([]uint64, []SourceType, error) {
	chainIDs := make([]uint64, 0, len(p.sources))
	for chainID := range p.sources {
		chainIDs = append(chainIDs, chainID)
	}

	return chainIDs, []SourceType{SourceTypeERC20, SourceTypeERC721, SourceTypeERC1155}, nil
}

func (p tokenListProvider) FetchTokenList(ctx context.Context, chainID uint64, source SourceType) (*TokenList, error) {
	m, ok := p.sources[chainID]
	if !ok {
		return nil, fmt.Errorf("no sources for chain %d", chainID)
	}
	url, ok := m[source]
	if !ok {
		return nil, nil
	}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	res, err := p.client.Do(req.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("fetching: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetching: %s", res.Status)
	}

	buf, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("reading body: %w", err)
	}

	var list TokenList
	if err := json.Unmarshal(buf, &list); err != nil {
		// failed to decode, likely because it doesn't follow standard token-list format,
		// and its just returning the ".tokens" part.
		list = TokenList{Name: fmt.Sprintf("%d", chainID), ChainID: chainID}

		tokens := list.Tokens
		err = json.Unmarshal(buf, &tokens)
		if err != nil {
			return nil, fmt.Errorf("decoding json: %w", err)
		}
		list.Tokens = tokens
	}

	return &list, nil
}
