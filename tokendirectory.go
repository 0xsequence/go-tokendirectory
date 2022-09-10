package tokendirectory

import (
	"context"
	"time"

	"github.com/0xsequence/go-sequence/lib/prototyp"
)

func NewTokenDirectoryFetcher(sources map[uint64][]string, updateFunc func(ctx context.Context, contractInfo *ContractInfo) error, durationForUpdates time.Duration) (*TokenDirectory, error) {
	if durationForUpdates == 0 {
		durationForUpdates = time.Minute * 15
	}
	if updateFunc == nil {
		updateFunc = func(ctx context.Context, contractInfo *ContractInfo) error {
			return nil
		}
	}
	lists := make(map[uint64]map[string]*TokenList)
	contracts := make(map[uint64]map[prototyp.Hash]*ContractInfo)
	for chainId, _ := range sources {
		lists[chainId] = make(map[string]*TokenList)
		contracts[chainId] = make(map[prototyp.Hash]*ContractInfo)
	}

	f := &TokenDirectory{
		ticker:     time.NewTicker(durationForUpdates),
		lists:      lists,
		contracts:  contracts,
		updateFunc: updateFunc,
		sources:    sources,
	}

	return f, nil
}
