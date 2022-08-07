package tokendirectory

import (
	"time"

	"github.com/0xsequence/go-sequence/lib/prototyp"
)

func NewTokenDirectoryFetcher(chainID uint64, sources []string, updateFunc func(contractInfo ContractInfo) error, durationForUpdates time.Duration) (*fetcher, error) {
	if durationForUpdates == 0 {
		durationForUpdates = time.Minute * 15
	}

	f := &fetcher{
		chainId:    chainID,
		ticker:     time.NewTicker(durationForUpdates),
		lists:      make(map[string]*TokenList),
		contracts:  make(map[prototyp.Hash]*ContractInfo),
		updateFunc: updateFunc,
		sources:    sources,
	}

	return f, nil
}
