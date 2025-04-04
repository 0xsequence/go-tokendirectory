package tokendirectory

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// TODO: I believe this provider may be useless and we don't use it anywhere..?
func NewSequenceProvider(baseURL string, client HTTPClient) (Provider, error) {
	if len(baseURL) == 0 {
		return nil, fmt.Errorf("baseURL is required")
	}

	if client == nil {
		client = http.DefaultClient
	}

	return sequenceProvider{
		id:      "sequence-builder-directory",
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

	baseURL := p.baseURL
	if baseURL[len(baseURL)-1] != '/' {
		baseURL = baseURL + "/"
	}
	url := baseURL + "token-directory/"

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
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
	baseURL := p.baseURL
	if baseURL[len(baseURL)-1] != '/' {
		baseURL = baseURL + "/"
	}
	url := fmt.Sprintf("%stoken-directory/%d/%s.json", baseURL, chainID, source)

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
