package tokendirectory

import (
	"context"
	"net/http"
)

var _DefaultMetadataSource = "https://metadata.sequence.app"

// Provider is a source of tokens, organized by chainID and sourceName.
type Provider interface {
	GetID() string
	GetConfig(ctx context.Context) (chainIDs []uint64, sources []SourceType, err error)
	FetchTokenList(ctx context.Context, chainID uint64, source SourceType) (*TokenList, error)
}

type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}
