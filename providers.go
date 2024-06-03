package tokendirectory

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"
)

var _DefaultMetadataSource = "https://metadata.sequence.app/"

// Provider is a source of tokens, organized by chainID and sourceName.
type Provider interface {
	GetID() string
	GetChainIDs() []uint64
	GetSources(chainID uint64) []string
	FetchTokenList(ctx context.Context, chainID uint64, sourceName string) (*TokenList, error)
}

func NewDefaultSequenceProvider(client *http.Client, metadataSource string, types ...SourceType) (Provider, error) {
	if client == nil {
		client = http.DefaultClient
	}

	return initializeSequenceMetadataProvider(client, metadataSource, types...)
}

func initializeSequenceMetadataProvider(client *http.Client, rootUrl string, types ...SourceType) (Provider, error) {
	respBody := struct {
		ChainIds []uint64 `json:"chainIds"`
		Types    []string `json:"sources"`
	}{}

	resp, err := http.Get(_DefaultMetadataSource + "/token-directory/")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch token-directory info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch token-directory info: %s", resp.Status)
	}

	if err := json.NewDecoder(resp.Body).Decode(&respBody); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	slices.Sort(respBody.ChainIds)

	sources := make(map[uint64]map[SourceType]string)
	for _, chainID := range respBody.ChainIds {
		sources[chainID] = make(map[SourceType]string)
		for _, t := range respBody.Types {
			sources[chainID][SourceType(strings.ToLower(t))] = fmt.Sprintf("%s/token-directory/%d/%s.json", rootUrl, chainID, t)
		}
	}

	if len(types) > 0 {
		for chainID, source := range sources {
			for _, t := range types {
				if _, ok := source[t]; !ok {
					delete(sources[chainID], t)
				}
			}
		}
	}

	return sequenceMetadataProvider{
		id:      "default",
		client:  client,
		rootUrl: rootUrl,
		sources: sources,
	}, nil
}

type sequenceMetadataProvider struct {
	id      string
	client  *http.Client
	rootUrl string
	sources map[uint64]map[SourceType]string
}

func (p sequenceMetadataProvider) GetID() string {
	return p.id

}

func (p sequenceMetadataProvider) GetChainIDs() []uint64 {
	chainIDs := make([]uint64, 0, len(p.sources))
	for chainID := range p.sources {
		chainIDs = append(chainIDs, chainID)
	}
	slices.Sort(chainIDs)
	return chainIDs
}

func (p sequenceMetadataProvider) GetSources(chainID uint64) []string {
	sources := make([]string, 0, len(p.sources[chainID]))
	for source := range p.sources[chainID] {
		sources = append(sources, source.String())
	}
	return sources
}

func (p sequenceMetadataProvider) FetchTokenList(ctx context.Context, chainID uint64, sourceName string) (*TokenList, error) {
	source, ok := p.sources[chainID]
	if !ok {
		return nil, fmt.Errorf("no sources for chain %d", chainID)
	}
	url, ok := source[SourceType(strings.ToLower(sourceName))]
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
