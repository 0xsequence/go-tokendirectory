package tokendirectory

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/0xsequence/go-sequence/lib/prototyp"
	"github.com/rs/zerolog/log"
)

type fetcher struct {
	chainId uint64

	ticker *time.Ticker

	sources   []string
	lists     map[string]*TokenList
	contracts map[prototyp.Hash]*ContractInfo

	updateFunc  func(contractInfo ContractInfo) error
	updateMu    sync.Mutex
	contractsMu sync.RWMutex
}

func (f *fetcher) Run(ctx context.Context) error {
	// TODO: consider running this as a goroutine, as long lists will take a long time
	f.updateLists()
	go f.listUpdater(ctx)
	return nil
}

func (f *fetcher) GetContractInfo(ctx context.Context, chainId uint64, contractAddr prototyp.Hash) (*ContractInfo, error) {
	if f.chainId != chainId {
		return nil, fmt.Errorf("chain ID mismatch: %v != %v", f.chainId, chainId)
	}

	f.contractsMu.RLock()
	defer f.contractsMu.RUnlock()

	if info, ok := f.contracts[contractAddr]; ok {
		return info, nil
	}

	return nil, nil
}

func (f *fetcher) listUpdater(ctx context.Context) {
	defer f.ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-f.ticker.C:
			f.updateLists()
		}
	}
}

func (f *fetcher) updateLists() {
	f.updateMu.Lock()
	defer f.updateMu.Unlock()

	seen := map[string]bool{}
	for _, source := range f.sources {
		tokenList, err := f.fetchTokenList(source)
		if err != nil {
			log.Warn().Err(err).Msgf("failed to fetch from source %q", source)
			continue
		}

		f.lists[source] = tokenList

		for i := range tokenList.Tokens {
			contractInfo := tokenList.Tokens[i]

			if contractInfo.Name == "" || contractInfo.Address == "" {
				continue
			}
			if contractInfo.ChainID != f.chainId {
				continue
			}

			if contractInfo.Type == "" {
				contractInfo.Type = strings.ToUpper(tokenList.TokenStandard)
			}

			contractInfo.Address = strings.ToLower(contractInfo.Address)

			if seen[contractInfo.Address] {
				// do not overwrite tokens that belong to a previous list
				continue
			}

			err = f.updateFunc(contractInfo)

			if err != nil {
				log.Warn().Err(err).Msgf("failed to execute update function for address %q chain %v", contractInfo.Address, contractInfo.ChainID)
			}
			f.contractsMu.Lock()
			f.contracts[prototyp.HashFromString(contractInfo.Address)] = &contractInfo
			f.contractsMu.Unlock()

			seen[contractInfo.Address] = true

		}
	}
}

func (f *fetcher) fetchTokenList(source string) (*TokenList, error) {
	log.Debug().Msgf("fetching tokens from source %q", source)

	httpClient := http.DefaultClient

	// pull from URL
	res, err := httpClient.Get(source)
	if err != nil {
		return nil, fmt.Errorf("failed fetching from %s: %w", source, err)
	}
	defer res.Body.Close()

	buf, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("failed reading body: %w", err)
	}

	var list TokenList
	if err := json.Unmarshal(buf, &list); err != nil {
		return nil, fmt.Errorf("failed decoding JSON: %w", err)
	}

	// normalize addresses
	tokenInfo := make([]ContractInfo, len(list.Tokens))
	for i, info := range list.Tokens {
		info.Address = strings.ToLower(info.Address)
		info.Extensions.OriginAddress = strings.ToLower(info.Extensions.OriginAddress)
		tokenInfo[i] = info
	}
	list.Tokens = tokenInfo

	return &list, nil
}
