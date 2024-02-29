package tokendirectory

import (
	"fmt"
	"net/http"
	"time"
)

type Option func(*TokenDirectory) error

func WithHTTPClient(client *http.Client) Option {
	return func(td *TokenDirectory) error {
		td.httpClient = client
		return nil
	}
}

func WithSources(sources ...Source) Option {
	return func(td *TokenDirectory) error {
		td.sources = sources
		return nil
	}
}

func WithUpdateFuncs(functions ...OnUpdateFunc) Option {
	return func(td *TokenDirectory) error {
		td.onUpdate = append(td.onUpdate, functions...)
		return nil
	}
}

func WithUpdateInterval(interval time.Duration) Option {
	return func(td *TokenDirectory) error {
		if interval < 1*time.Minute {
			return fmt.Errorf("updateInterval must be greater then 1 minute")
		}
		td.updateInterval = interval
		return nil
	}
}
