package main

import (
	"context"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/0xsequence/go-tokendirectory"
	"github.com/lmittmann/tint"
)

var debug = flag.Bool("debug", false, "enable debug logging")

func main() {
	flag.Parse()

	level := slog.LevelInfo
	if *debug {
		level = slog.LevelDebug
	}
	logger := slog.New(tint.NewHandler(os.Stderr, &tint.Options{Level: level}))

	updateFunc := func(ctx context.Context, chainID uint64, list []tokendirectory.ContractInfo) {
		logger := logger.With(slog.Uint64("chainID", chainID))
		for _, c := range list {
			logger.With(slog.String("address", c.Address), slog.String("name", c.Name)).Info("updated info")
		}
	}

	logger.Info("go-tokendirectory example starting...")

	provider, err := tokendirectory.NewSequenceProvider(http.DefaultClient, "https://dev-metadata.sequence.app/")
	if err != nil {
		panic(err)
	}

	options := []tokendirectory.Option{
		tokendirectory.WithUpdateFuncs(updateFunc),
		tokendirectory.WithUpdateInterval(time.Minute),
		tokendirectory.WithLogger(logger),
		tokendirectory.WithProviders(provider),
		tokendirectory.WithSources(tokendirectory.SourceTypeERC1155),
	}

	tokenDirectory, err := tokendirectory.NewTokenDirectory(options...)
	if err != nil {
		panic(err)
	}

	go func() {
		err := tokenDirectory.Run(context.Background())
		if err != nil {
			panic(err)
		}
	}()

	time.Sleep(time.Second * 150)
	tokenDirectory.Stop()
}
