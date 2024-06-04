package tokendirectory

import (
	"fmt"
	"log/slog"
	"time"
)

type Option func(*TokenDirectory) error

func WithLogger(logger *slog.Logger) Option {
	return func(td *TokenDirectory) error {
		td.log = logger
		return nil
	}
}

func WithProviders(providers ...Provider) Option {
	return func(td *TokenDirectory) error {
		if len(providers) == 0 {
			return fmt.Errorf("no provider specified")
		}
		for _, p := range providers {
			if _, ok := td.providers[p.GetID()]; ok {
				return fmt.Errorf("provider %q already exists", p.GetID())
			}
			td.providers[p.GetID()] = p
		}
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

func WithChainIDs(chainIDs ...uint64) Option {
	return func(td *TokenDirectory) error {
		td.chainIDs = chainIDs
		return nil
	}
}

func WithSources(sources ...SourceType) Option {
	return func(td *TokenDirectory) error {
		td.sources = sources
		return nil
	}
}
