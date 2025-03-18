package tokendirectory

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path"
)

func NewSequenceProvider(baseURL string, client HTTPClient) (Provider, error) {
	if client == nil {
		client = http.DefaultClient
	}

	return sequenceProvider{
		id:      "sequence-token-directory",
		client:  client,
		baseURL: baseURL,
	}, nil
}

type sequenceProvider struct {
	id      string
	client  HTTPClient
	baseURL string
}

func (p sequenceProvider) GetConfig(ctx context.Context) (chainIDs []uint64, sources []SourceType, err error) {
	respBody := struct {
		ChainIds []uint64     `json:"chainIds"`
		Types    []SourceType `json:"sources"`
	}{}

	req, err := http.NewRequestWithContext(ctx, "GET", path.Join(p.baseURL, "/token-directory/"), nil)
	if err != nil {
		return nil, nil, fmt.Errorf("creating request: %w", err)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("info: %s", resp.Status)
	}

	if err := json.NewDecoder(resp.Body).Decode(&respBody); err != nil {
		return nil, nil, fmt.Errorf("decode: %w", err)
	}

	return respBody.ChainIds, respBody.Types, nil
}

func (p sequenceProvider) GetID() string {
	return p.id
}

func (p sequenceProvider) FetchTokenList(ctx context.Context, chainID uint64, source SourceType) (*TokenList, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/token-directory/%d/%s.json", p.baseURL, chainID, source), nil)
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
