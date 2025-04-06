package main

import (
	"context"
	"fmt"

	"github.com/0xsequence/go-tokendirectory"
	"github.com/davecgh/go-spew/spew"
)

func main() {
	td := tokendirectory.NewTokenDirectory(tokendirectory.Options{ChainIDs: []uint64{1, 137}}) //, SkipExternalTokenLists: true})
	// td := tokendirectory.NewTokenDirectory(tokendirectory.Options{TokenListURLs: []string{"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/mainnet/erc20.json"}})
	// td := tokendirectory.NewTokenDirectory(tokendirectory.Options{IncludeDeprecated: true})
	index, err := td.FetchIndex(context.Background())
	if err != nil {
		panic(err)
	}
	spew.Dump(index)

	// tokenLists, err := td.FetchTokenLists(context.Background(), []uint64{1}...)
	// if err != nil {
	// 	panic(err)
	// }
	// spew.Dump(tokenLists)

	// .. TODO: .. need DiffIndex method.. to take two indexes.. and get a diff..

	// maybe we do stuff like GetIndexDiff(context.Background(), index, index2)

	//--

	fmt.Println("----")

	index2, err := td.FetchIndex(context.Background()) //, tokendirectory.IndexFilter{ChainIDs: []uint64{1}}) //, External: true, Deprecated: true})
	if err != nil {
		panic(err)
	}
	spew.Dump(index2)

	tokenLists, err := td.FetchTokenLists(context.Background(), index2)
	if err != nil {
		panic(err)
	}
	fmt.Println("=> len", len(tokenLists))

	tokenLists, err = td.FetchTokenLists(context.Background(), index2)
	if err != nil {
		panic(err)
	}
	fmt.Println("=> len", len(tokenLists[1]))

	// tokenLists, err := td.FetchChainTokenLists(context.Background(), 1)
	// if err != nil {
	// 	panic(err)
	// }
	// // spew.Dump(tokenLists)
	// fmt.Println("=> len", len(tokenLists))

	// externalTokenLists, err := td.FetchExternalTokenLists(context.Background())
	// if err != nil {
	// 	panic(err)
	// }
	// // spew.Dump(externalTokenLists)
	// fmt.Println("=> len", len(externalTokenLists))

}
