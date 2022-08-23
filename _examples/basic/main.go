package main

import (
	"context"
	"fmt"
	"time"

	"github.com/0xsequence/go-tokendirectory"
)

func main() {
	fmt.Println("Starting Program...")

	updateFunc := func(ctx context.Context, contractInfo *tokendirectory.ContractInfo) error {
		fmt.Printf("updating %v\n", contractInfo.Address)
		return nil
	}

	tokenDirectoryFetcher, err := tokendirectory.NewTokenDirectoryFetcher(1, tokendirectory.DefaultTokenDirectorySources, updateFunc, time.Second*30)
	if err != nil {
		panic(err)
	}

	ctx := context.Background()
	err = tokenDirectoryFetcher.Run(ctx)
	if err != nil {
		panic(err)
	}
	time.Sleep(time.Second * 60)
	// stops the fetcher
	ctx.Done()
}
