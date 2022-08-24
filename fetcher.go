package tokendirectory

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/0xsequence/go-sequence/lib/prototyp"
	"github.com/rs/zerolog/log"
)

type TokenDirectory struct {
	ticker *time.Ticker

	sources   map[uint64][]string
	lists     map[uint64]map[string]*TokenList
	contracts map[uint64]map[prototyp.Hash]*ContractInfo

	updateFunc  func(ctx context.Context, contractInfo *ContractInfo) error
	updateMu    sync.Mutex
	contractsMu sync.RWMutex
}

func (f *TokenDirectory) Run(ctx context.Context) error {
	wg := &sync.WaitGroup{}
	for chainID := range f.sources {
		wg.Add(1)
		go func(chainID uint64) {
			defer wg.Done()
			// TODO: consider running this as a goroutine, as long lists will take a long time
			f.updateLists(ctx, chainID)
			go f.listUpdater(ctx, chainID)
		}(chainID)
	}
	wg.Wait()
	return nil
}

func (f *TokenDirectory) GetContractInfo(ctx context.Context, chainId uint64, contractAddr prototyp.Hash) (*ContractInfo, error) {
	if _, ok := f.contracts[chainId]; !ok {
		return nil, fmt.Errorf("chain ID not supported: %v", chainId)
	}

	f.contractsMu.RLock()
	defer f.contractsMu.RUnlock()

	if info, ok := f.contracts[chainId][contractAddr]; ok {
		return info, nil
	}

	return nil, errors.New("contract not found")
}

func (f *TokenDirectory) listUpdater(ctx context.Context, chainID uint64) {
	defer f.ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-f.ticker.C:
			f.updateLists(ctx, chainID)
		}
	}
}

func (f *TokenDirectory) updateLists(ctx context.Context, chainID uint64) {
	f.updateMu.Lock()
	defer f.updateMu.Unlock()

	seen := map[string]bool{}
	f.contractsMu.Lock()
	if _, ok := f.lists[chainID]; !ok {
		f.lists[chainID] = make(map[string]*TokenList)
	}
	if _, ok := f.contracts[chainID]; !ok {
		f.contracts[chainID] = make(map[prototyp.Hash]*ContractInfo)
	}
	f.contractsMu.Unlock()

	for _, source := range f.sources[chainID] {
		tokenList, err := f.fetchTokenList(source)
		if err != nil {
			log.Warn().Err(err).Msgf("failed to fetch from source %q", source)
			continue
		}

		f.lists[chainID][source] = tokenList

		for i := range tokenList.Tokens {
			contractInfo := tokenList.Tokens[i]

			if contractInfo.Name == "" || contractInfo.Address == "" {
				continue
			}
			if contractInfo.ChainID != chainID {
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

			// this is a function that will be called when the contract info is updated
			// run it as a goroutine so that it doesn't block the update loop
			go f.updateFunc(ctx, &contractInfo)

			if err != nil {
				log.Warn().Err(err).Msgf("failed to execute update function for address %q chain %v", contractInfo.Address, contractInfo.ChainID)
			}
			f.contractsMu.Lock()
			f.contracts[chainID][prototyp.HashFromString(contractInfo.Address)] = &contractInfo
			f.contractsMu.Unlock()
			seen[contractInfo.Address] = true
		}
	}
}

func (f *TokenDirectory) fetchTokenList(source string) (*TokenList, error) {
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
