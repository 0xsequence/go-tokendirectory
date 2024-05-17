package tokendirectory

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"slices"
)

// Provider is a source of tokens, organized by chainID and sourceName.
type Provider interface {
	GetID() string
	GetChainIDs() []uint64
	GetSources(chainID uint64) []string
	FetchTokenList(ctx context.Context, chainID uint64, sourceName string) (*TokenList, error)
}

func NewProvider(client *http.Client, types ...SourceType) Provider {
	if client == nil {
		client = http.DefaultClient
	}

	sources := _DefaultSources
	if len(types) > 0 {
		sources = make(map[uint64]map[SourceType]string, len(_DefaultSources))
		for chainID, source := range _DefaultSources {
			sources[chainID] = make(map[SourceType]string, len(types))
			for _, t := range types {
				if url, ok := source[t]; ok {
					sources[chainID][t] = url
				}
			}
		}
	}

	return urlListProvider{
		client:  client,
		sources: sources,
	}
}

type urlListProvider struct {
	id      string
	client  *http.Client
	sources map[uint64]map[SourceType]string
}

func (p urlListProvider) GetID() string {
	return p.id

}

func (p urlListProvider) GetChainIDs() []uint64 {
	chainIDs := make([]uint64, 0, len(p.sources))
	for chainID := range p.sources {
		chainIDs = append(chainIDs, chainID)
	}
	slices.Sort(chainIDs)
	return chainIDs
}

func (p urlListProvider) GetSources(chainID uint64) []string {
	sources := make([]string, 0, len(p.sources[chainID]))
	for source := range p.sources[chainID] {
		sources = append(sources, source.String())
	}
	return sources
}

func (p urlListProvider) FetchTokenList(ctx context.Context, chainID uint64, sourceName string) (*TokenList, error) {
	source, ok := p.sources[chainID]
	if !ok {
		return nil, fmt.Errorf("no sources for chain %d", chainID)
	}
	url, ok := source[SourceType(sourceName)]
	if !ok {
		return nil, fmt.Errorf("no source %q for chain %d", sourceName, chainID)
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
