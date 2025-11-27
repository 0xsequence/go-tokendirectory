package main

import (
	"context"
	"fmt"

	"github.com/0xsequence/go-tokendirectory"
)

func main() {
	// example1()
	example2()
}

func example1() {
	td := tokendirectory.NewTokenDirectory(tokendirectory.Options{ChainIDs: []uint64{1, 137}}) //, SkipExternalTokenLists: true})
	// td := tokendirectory.NewTokenDirectory(tokendirectory.Options{TokenListURLs: []string{"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/mainnet/erc20.json"}})
	// td := tokendirectory.NewTokenDirectory(tokendirectory.Options{IncludeDeprecated: true})

	index, err := td.FetchIndex(context.Background(), tokendirectory.IndexFilter{ChainIDs: []uint64{1}})
	if err != nil {
		panic(err)
	}
	// spew.Dump(index)

	fmt.Println("")
	fmt.Println("")

	index2, err := td.FetchIndex(context.Background())
	if err != nil {
		panic(err)
	}
	// spew.Dump(index2)

	tokenLists, err := td.FetchTokenLists(context.Background(), index2)
	if err != nil {
		panic(err)
	}
	fmt.Println("=> len", len(tokenLists))

	// cached
	tokenLists, err = td.FetchTokenLists(context.Background(), index2)
	if err != nil {
		panic(err)
	}
	fmt.Println("=> len", len(tokenLists[1]))

	fmt.Println("")
	fmt.Println("")

	fmt.Println("DIFF:")

	diff := tokendirectory.DiffIndex(index, index2)
	_ = diff
}

func example2() {
	td := tokendirectory.NewTokenDirectory(tokendirectory.Options{
		ChainIDs:  []uint64{0, 1, 137, 747474},
		OnlyERC20: true,
		//, SkipExternalTokenLists: true,
	})

	index, err := td.FetchIndex(context.Background())
	if err != nil {
		panic(err)
	}

	tokenLists, err := td.FetchTokenLists(context.Background(), index)
	if err != nil {
		panic(err)
	}
	fmt.Println("=> len", len(tokenLists))
	for chainID, list := range tokenLists {
		fmt.Println("   - chainID:", chainID, " tokens:", len(list))
		for _, tokenList := range list {
			fmt.Printf("      - %s %s %d %s\n", tokenList.Name, tokenList.TokenStandard, len(tokenList.Tokens), tokenList.TokenListURL)
		}
	}

	// print the token contract info
	contractInfo, err := td.FetchTokenContractInfo(context.Background(), index)
	if err != nil {
		panic(err)
	}
	fmt.Println("=> contract info len", len(contractInfo))
	for chainID, contracts := range contractInfo {
		fmt.Println("   - chainID:", chainID, " contracts:", len(contracts))
		for _, contract := range contracts {
			fmt.Printf("      - %d %s %s %d\n", contract.ChainID, contract.Address, contract.Name, contract.Extensions.FeatureIndex)
		}
	}
}
