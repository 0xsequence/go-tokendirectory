package main

import (
	"context"
	"fmt"
	"time"

	"github.com/0xsequence/go-tokendirectory"
)

func main() {
	fmt.Println("go-tokendirectory example starting..")

	updateFunc := func(ctx context.Context, chainID uint64, contractInfoList []tokendirectory.ContractInfo) {
		for _, contractInfo := range contractInfoList {
			fmt.Printf("updating %v\n", contractInfo.Address)
		}
	}

	tokenDirectory, err := tokendirectory.NewTokenDirectory(tokendirectory.DefaultSources, time.Minute*1, updateFunc)
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
