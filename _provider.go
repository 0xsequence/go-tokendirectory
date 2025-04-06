package tokendirectory

import (
	"context"
	"net/http"
)

// Provider is a source of tokens, organized by chainID and sourceName.
type Provider interface {
	FetchTokenList(ctx context.Context, chainID uint64, source SourceType) (*TokenList, error)
}

type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}
