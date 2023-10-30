package tokendirectory

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/0xsequence/go-sequence/lib/prototyp"
	"github.com/rs/zerolog/log"
)

func NewTokenDirectory(sources Sources, updateInterval time.Duration, onUpdate ...onUpdateFunc) (*TokenDirectory, error) {
	if updateInterval == 0 {
		// default update every 15 minutes
		updateInterval = time.Minute * 15
	}
	if updateInterval < 1*time.Minute {
		return nil, fmt.Errorf("updateInterval must be greater then 1 minute")
	}

	var updateFunc onUpdateFunc
	if len(onUpdate) > 0 {
		updateFunc = onUpdate[0]
	}

	lists := make(map[uint64]map[string]*TokenList)
	contracts := make(map[uint64]map[prototyp.Hash]ContractInfo)

	for chainId, _ := range sources {
		lists[chainId] = make(map[string]*TokenList)
		contracts[chainId] = make(map[prototyp.Hash]ContractInfo)
	}

	f := &TokenDirectory{
		sources:        sources,
		lists:          lists,
		contracts:      contracts,
		httpClient:     http.DefaultClient,
		updateInterval: updateInterval,
		onUpdate:       updateFunc,
	}

	return f, nil
}

type TokenDirectory struct {
	sources Sources
	lists   map[uint64]map[string]*TokenList

	contracts   map[uint64]map[prototyp.Hash]ContractInfo
	contractsMu sync.RWMutex

	updateInterval time.Duration
	onUpdate       onUpdateFunc
	updateMu       sync.Mutex

	httpClient *http.Client

	ctx     context.Context
	ctxStop context.CancelFunc
	running int32
}

type onUpdateFunc func(ctx context.Context, chainID uint64, contractInfoList []ContractInfo)

// SetHttpClient sets the http client used to fetch token-lists from remote sources.
func (f *TokenDirectory) SetHttpClient(client *http.Client) {
	f.httpClient = client
}

// Run starts the token directory fetcher. This method will block and poll in the current
// go-routine. You'll be responsible for calling the Run method in your own gorutine.
func (f *TokenDirectory) Run(ctx context.Context) error {
	if f.IsRunning() {
		return fmt.Errorf("tokendirectory: already running")
	}

	f.ctx, f.ctxStop = context.WithCancel(ctx)

	atomic.StoreInt32(&f.running, 1)
	defer atomic.StoreInt32(&f.running, 0)

	// Initial source fetch
	f.updateSources(f.ctx)

	// Fetch on interval
	for {
		select {
		case <-f.ctx.Done():
			return nil
		case <-time.After(f.updateInterval):
			f.updateSources(f.ctx)
		}
	}
}

func (f *TokenDirectory) Stop() {
	log.Info().Msgf("tokendirectory: stop")
	f.ctxStop()
}

func (f *TokenDirectory) IsRunning() bool {
	return atomic.LoadInt32(&f.running) == 1
}

func (f *TokenDirectory) updateSources(ctx context.Context) error {
	wg := &sync.WaitGroup{}
	for chainID := range f.sources {
		wg.Add(1)
		go func(chainID uint64) {
			defer wg.Done()
			f.updateChainSource(ctx, chainID)
		}(chainID)
	}
	wg.Wait()
	return nil
}

func (f *TokenDirectory) updateChainSource(ctx context.Context, chainID uint64) {
	f.updateMu.Lock()
	defer f.updateMu.Unlock()

	updatedContractInfo := []ContractInfo{}
	seen := map[string]bool{}

	f.contractsMu.Lock()
	if _, ok := f.lists[chainID]; !ok {
		f.lists[chainID] = make(map[string]*TokenList)
	}
	if _, ok := f.contracts[chainID]; !ok {
		f.contracts[chainID] = make(map[prototyp.Hash]ContractInfo)
	}
	f.contractsMu.Unlock()

	for _, source := range f.sources[chainID] {
		tokenList, err := f.fetchTokenList(chainID, source)
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

			// keep track of contract info which has been updated
			if f.onUpdate != nil {
				updatedContractInfo = append(updatedContractInfo, contractInfo)
			}

			if err != nil {
				log.Warn().Err(err).Msgf("failed to execute update function for address %q chain %v", contractInfo.Address, contractInfo.ChainID)
			}
			f.contractsMu.Lock()
			f.contracts[chainID][prototyp.HashFromString(contractInfo.Address)] = contractInfo
			f.contractsMu.Unlock()
			seen[contractInfo.Address] = true
		}
	}

	if f.onUpdate != nil {
		if len(updatedContractInfo) > 0 {
			go f.onUpdate(ctx, chainID, updatedContractInfo)
		}
	}
}

func (f *TokenDirectory) fetchTokenList(chainID uint64, source string) (*TokenList, error) {
	log.Debug().Msgf("fetching tokens from source %q", source)

	// pull from URL
	res, err := f.httpClient.Get(source)
	if err != nil {
		return nil, fmt.Errorf("failed fetching from %s: %w", source, err)
	}
	defer res.Body.Close()

	buf, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("failed reading body: %w", err)
	}

	var list TokenList
	if err := json.Unmarshal(buf, &list); err != nil {
		// failed to decode, likely because it doesn't follow standard token-list format,
		// and its just returning the ".tokens" part.
		list = TokenList{Name: fmt.Sprintf("%d", chainID), ChainID: chainID}

		tokens := list.Tokens
		err = json.Unmarshal(buf, &tokens)
		if err != nil {
			return nil, fmt.Errorf("failed decoding JSON: %w", err)
		}
		list.Tokens = tokens
	}

	// normalize addresses
	tokenInfo := make([]ContractInfo, len(list.Tokens))
	for i, info := range list.Tokens {
		info.Address = strings.ToLower(info.Address)
		info.Extensions.OriginAddress = strings.ToLower(info.Extensions.OriginAddress)
		info.Type = strings.ToUpper(list.TokenStandard)
		// add the token-directory verification stamp
		info.Extensions.Verified = !info.Extensions.Blacklist
		verifiedBy := "token-directory"
		info.Extensions.VerifiedBy = &verifiedBy
		tokenInfo[i] = info
	}
	list.Tokens = tokenInfo

	return &list, nil
}

func (f *TokenDirectory) GetContractInfo(ctx context.Context, chainId uint64, contractAddr prototyp.Hash) (ContractInfo, bool, error) {
	if _, ok := f.contracts[chainId]; !ok {
		return ContractInfo{}, false, fmt.Errorf("chain ID not supported: %v", chainId)
	}

	f.contractsMu.RLock()
	defer f.contractsMu.RUnlock()

	if info, ok := f.contracts[chainId][contractAddr]; ok {
		return info, true, nil
	}

	return ContractInfo{}, false, errors.New("contract not found")
}

func (f *TokenDirectory) GetNetworks(ctx context.Context) ([]uint64, error) {
	chainIDs := make([]uint64, 0, len(f.lists))
	for chainID := range f.lists {
		list, err := f.GetTokens(ctx, chainID)
		if err != nil {
			return nil, err
		}
		if len(list) == 0 {
			continue
		}
		chainIDs = append(chainIDs, chainID)
	}
	return chainIDs, nil
}

func (f *TokenDirectory) GetAllTokens(ctx context.Context) ([]ContractInfo, error) {
	var tokens []ContractInfo
	for chainID := range f.lists {
		list, err := f.GetTokens(ctx, chainID)
		if err != nil {
			return nil, err
		}
		tokens = append(tokens, list...)
	}
	return tokens, nil
}

func (f *TokenDirectory) GetTokens(ctx context.Context, chainID uint64) ([]ContractInfo, error) {
	tokens := make([]ContractInfo, 0, len(f.lists[chainID]))
	for _, list := range f.lists[chainID] {
		tokens = append(tokens, list.Tokens...)
	}
	return tokens, nil
}
