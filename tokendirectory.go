package tokendirectory

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/0xsequence/go-sequence/lib/prototyp"
	"github.com/rs/zerolog/log"
)

func NewTokenDirectory(options ...Option) (*TokenDirectory, error) {
	dir := &TokenDirectory{
		lists:     make(map[uint64]map[string]*TokenList),
		contracts: make(map[uint64]map[prototyp.Hash]ContractInfo),
	}

	for _, option := range options {
		if err := option(dir); err != nil {
			return nil, err
		}
	}

	if dir.log == nil {
		dir.log = slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
	}
	if dir.httpClient == nil {
		dir.httpClient = http.DefaultClient
	}
	if dir.updateInterval == 0 {
		dir.updateInterval = time.Minute * 15
	}
	if len(dir.providers) == 0 {
		dir.providers = append(dir.providers, &defaultProvider{client: dir.httpClient})
	}

	// initialize the token lists
	for _, source := range dir.providers {
		for _, chainId := range source.GetChainIDs() {
			dir.lists[chainId] = make(map[string]*TokenList)
			dir.contracts[chainId] = make(map[prototyp.Hash]ContractInfo)
		}
	}

	return dir, nil
}

type TokenDirectory struct {
	log       *slog.Logger
	providers []Provider
	lists     map[uint64]map[string]*TokenList

	contracts   map[uint64]map[prototyp.Hash]ContractInfo
	contractsMu sync.RWMutex

	updateInterval time.Duration
	onUpdate       []OnUpdateFunc
	updateMu       sync.Mutex

	httpClient *http.Client

	ctx     context.Context
	ctxStop context.CancelFunc
	running int32
}

type OnUpdateFunc func(ctx context.Context, chainID uint64, contractInfoList []ContractInfo)

// Run starts the token directory fetcher. This method will block and poll in the current
// go-routine. You'll be responsible for calling the Run method in your own gorutine.
func (t *TokenDirectory) Run(ctx context.Context) error {
	if t.IsRunning() {
		return fmt.Errorf("tokendirectory: already running")
	}

	t.ctx, t.ctxStop = context.WithCancel(ctx)

	atomic.StoreInt32(&t.running, 1)
	defer atomic.StoreInt32(&t.running, 0)

	// Initial source fetch
	t.updateSources(t.ctx)

	// Fetch on interval
	for {
		select {
		case <-t.ctx.Done():
			return nil
		case <-time.After(t.updateInterval):
			t.updateSources(t.ctx)
		}
	}
}

func (t *TokenDirectory) Stop() {
	log.Info().Msgf("tokendirectory: stop")
	t.ctxStop()
}

func (t *TokenDirectory) IsRunning() bool {
	return atomic.LoadInt32(&t.running) == 1
}

func (t *TokenDirectory) updateSources(ctx context.Context) error {
	wg := &sync.WaitGroup{}
	for _, provider := range t.providers {
		for _, chainID := range provider.GetChainIDs() {
			for _, source := range provider.GetSources(chainID) {
				wg.Add(1)
				go func(provider Provider, chainID uint64, source string) {
					defer wg.Done()
					t.updateProvider(ctx, provider, chainID, source)
				}(provider, chainID, source)
			}
		}
	}
	wg.Wait()
	return nil
}

func (t *TokenDirectory) updateProvider(ctx context.Context, provider Provider, chainID uint64, source string) {
	t.updateMu.Lock()
	defer t.updateMu.Unlock()

	updatedContractInfo := []ContractInfo{}
	seen := map[string]bool{}

	t.contractsMu.Lock()
	if _, ok := t.lists[chainID]; !ok {
		t.lists[chainID] = make(map[string]*TokenList)
	}
	if _, ok := t.contracts[chainID]; !ok {
		t.contracts[chainID] = make(map[prototyp.Hash]ContractInfo)
	}
	t.contractsMu.Unlock()

	providerID := provider.GetID()

	logger := t.log.With(
		slog.String("providerID", providerID),
		slog.Uint64("chainID", chainID),
		slog.String("source", source),
	)

	logger.Debug("fetching token list")
	tokenList, err := provider.FetchTokenList(ctx, chainID, source)
	if err != nil {
		logger.With(slog.Any("err", err)).Error("failed to fetch token list")
		return
	}

	t.lists[chainID][providerID] = tokenList

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
		if t.onUpdate != nil {
			updatedContractInfo = append(updatedContractInfo, contractInfo)
		}

		t.contractsMu.Lock()
		t.contracts[chainID][prototyp.HashFromString(contractInfo.Address)] = contractInfo
		t.contractsMu.Unlock()
		seen[contractInfo.Address] = true
	}

	if t.onUpdate != nil {
		if len(updatedContractInfo) > 0 {
			for i := range t.onUpdate {
				go t.onUpdate[i](ctx, chainID, updatedContractInfo)
			}
		}
	}
}

func (t *TokenDirectory) GetContractInfo(ctx context.Context, chainId uint64, contractAddr prototyp.Hash) (ContractInfo, bool, error) {
	if _, ok := t.contracts[chainId]; !ok {
		return ContractInfo{}, false, fmt.Errorf("chain ID not supported: %v", chainId)
	}

	t.contractsMu.RLock()
	defer t.contractsMu.RUnlock()

	if info, ok := t.contracts[chainId][contractAddr]; ok {
		return info, true, nil
	}

	return ContractInfo{}, false, errors.New("contract not found")
}

func (t *TokenDirectory) GetNetworks(ctx context.Context) ([]uint64, error) {
	chainIDs := make([]uint64, 0, len(t.lists))
	for chainID := range t.lists {
		list, err := t.GetTokens(ctx, chainID)
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

func (t *TokenDirectory) GetAllTokens(ctx context.Context) ([]ContractInfo, error) {
	var tokens []ContractInfo
	for chainID := range t.lists {
		list, err := t.GetTokens(ctx, chainID)
		if err != nil {
			return nil, err
		}
		tokens = append(tokens, list...)
	}
	return tokens, nil
}

func (t *TokenDirectory) GetTokens(ctx context.Context, chainID uint64) ([]ContractInfo, error) {
	tokens := make([]ContractInfo, 0, len(t.lists[chainID]))
	for _, list := range t.lists[chainID] {
		tokens = append(tokens, list.Tokens...)
	}
	return tokens, nil
}
