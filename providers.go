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

type defaultProvider struct {
	client *http.Client
}

func (defaultProvider) GetID() string {
	return "token-directory"

}

func (defaultProvider) GetChainIDs() []uint64 {
	chainIDs := make([]uint64, 0, len(defaultSources))
	for chainID := range defaultSources {
		chainIDs = append(chainIDs, chainID)
	}
	slices.Sort(chainIDs)
	return chainIDs
}

func (defaultProvider) GetSources(chainID uint64) []string {
	sources := make([]string, 0, len(defaultSources[chainID]))
	for source := range defaultSources[chainID] {
		sources = append(sources, source)
	}
	return sources
}

func (p defaultProvider) FetchTokenList(ctx context.Context, chainID uint64, sourceName string) (*TokenList, error) {
	req, err := http.NewRequest("GET", defaultSources[chainID][sourceName], nil)
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
