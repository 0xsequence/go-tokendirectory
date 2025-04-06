package tokendirectory

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/0xsequence/go-sequence/lib/prototyp"
)

func NewTokenDirectory(options ...Option) (*TokenDirectory, error) {
	dir := &TokenDirectory{
		providers: make(map[string]Provider),
		lists:     make(map[uint64]map[SourceType]*TokenList),
		contracts: make(map[uint64]map[prototyp.Hash]ContractInfo),
	}

	for _, option := range options {
		if err := option(dir); err != nil {
			return nil, err
		}
	}

	if dir.updateInterval == 0 {
		dir.updateInterval = time.Minute * 15
	}
	if len(dir.providers) == 0 {
		// seqProvider, err := NewSequenceProvider(_DefaultMetadataSource, http.DefaultClient)
		// if err != nil {
		// 	return nil, err
		// }
		// dir.providers = map[string]Provider{"default": seqProvider}
	}
	if dir.log == nil {
		// Use a no-op logger by default
		dir.log = slog.New(slog.NewTextHandler(io.Discard, nil))
	}

	return dir, nil
}

type TokenDirectory struct {
	log       *slog.Logger
	providers map[string]Provider
	lists     map[uint64]map[SourceType]*TokenList

	contracts   map[uint64]map[prototyp.Hash]ContractInfo
	contractsMu sync.RWMutex

	updateInterval time.Duration
	onUpdate       []OnUpdateFunc
	updateMu       sync.Mutex

	chainIDs []uint64
	sources  []SourceType

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
	if err := t.updateSources(t.ctx); err != nil {
		return fmt.Errorf("initial source fetch: %w", err)
	}

	// Fetch on interval
	for {
		select {
		case <-t.ctx.Done():
			return nil
		case <-time.After(t.updateInterval):
			if err := t.updateSources(t.ctx); err != nil {
				t.log.Error("failed to update sources", slog.Any("err", err))
				// Continue running despite errors
			}
		}
	}
}

func (t *TokenDirectory) Stop() {
	t.log.Info("tokendirectory: stop")
	t.ctxStop()
}

func (t *TokenDirectory) IsRunning() bool {
	return atomic.LoadInt32(&t.running) == 1
}

func (t *TokenDirectory) updateSources(ctx context.Context) error {
	wg := &sync.WaitGroup{}
	for _, provider := range t.providers {
		chainIDs, sources, err := provider.GetConfig(ctx)
		if err != nil {
			return fmt.Errorf("get config: %w", err)
		}
		for _, chainID := range chainIDs {
			if len(t.chainIDs) > 0 && !slices.Contains(t.chainIDs, chainID) {
				continue
			}
			for _, source := range sources {
				if len(t.sources) > 0 && !slices.Contains(t.sources, source) {
					continue
				}
				wg.Add(1)
				go func(provider Provider, chainID uint64, source SourceType) {
					defer wg.Done()
					t.updateProvider(ctx, provider, chainID, source)
				}(provider, chainID, source)
			}
		}
	}
	wg.Wait()
	return nil
}

func (t *TokenDirectory) updateProvider(ctx context.Context, provider Provider, chainID uint64, source SourceType) {
	t.updateMu.Lock()
	var err error
	defer func() {
		t.updateMu.Unlock()
		if t.log != nil {
			logger := t.log.With(
				slog.String("provider", provider.GetID()),
				slog.Uint64("chainId", chainID),
				slog.String("source", source.String()),
			)
			if err != nil {
				logger.Error("failed to update provider", slog.Any("err", err))
				return
			}
			logger.Debug("updated provider")
		}
	}()

	// Initialize maps if they don't exist
	t.ensureMapsInitialized(chainID)

	// Fetch token list
	tokenList, err := provider.FetchTokenList(ctx, chainID, source)
	if err != nil || tokenList == nil {
		return
	}
	normalizeTokens(provider, tokenList)

	t.lists[chainID][source] = tokenList

	// Process and store tokens
	updatedContractInfo := t.processTokenList(chainID, tokenList)

	// Trigger update callbacks
	t.triggerUpdateCallbacks(ctx, chainID, updatedContractInfo)
}

// ensureMapsInitialized ensures the maps for a chain ID are initialized
func (t *TokenDirectory) ensureMapsInitialized(chainID uint64) {
	t.contractsMu.Lock()
	defer t.contractsMu.Unlock()

	if _, ok := t.lists[chainID]; !ok {
		t.lists[chainID] = make(map[SourceType]*TokenList)
	}
	if _, ok := t.contracts[chainID]; !ok {
		t.contracts[chainID] = make(map[prototyp.Hash]ContractInfo)
	}
}

// processTokenList processes the token list and stores tokens in the contract map
func (t *TokenDirectory) processTokenList(chainID uint64, tokenList *TokenList) []ContractInfo {
	updatedContractInfo := []ContractInfo{}
	seen := map[string]struct{}{}

	for _, token := range tokenList.Tokens {
		if !t.isValidToken(token, chainID) {
			continue
		}

		if token.Type == "" {
			token.Type = strings.ToUpper(tokenList.TokenStandard)
		}

		token.Address = strings.ToLower(token.Address)

		if _, ok := seen[token.Address]; ok {
			// do not overwrite tokens that belong to a previous list
			continue
		}

		// keep track of contract info which has been updated
		if t.onUpdate != nil {
			updatedContractInfo = append(updatedContractInfo, token)
		}

		t.contractsMu.Lock()
		t.contracts[chainID][prototyp.HashFromString(token.Address)] = token
		t.contractsMu.Unlock()
		seen[token.Address] = struct{}{}
	}

	return updatedContractInfo
}

// isValidToken checks if a token is valid for inclusion
func (t *TokenDirectory) isValidToken(token ContractInfo, chainID uint64) bool {
	return token.Name != "" && token.Address != "" && token.ChainID == chainID
}

// triggerUpdateCallbacks triggers the registered update callbacks
func (t *TokenDirectory) triggerUpdateCallbacks(ctx context.Context, chainID uint64, updatedContractInfo []ContractInfo) {
	if t.onUpdate != nil && len(updatedContractInfo) > 0 {
		for i := range t.onUpdate {
			go t.onUpdate[i](ctx, chainID, updatedContractInfo)
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
	t.contractsMu.RLock()
	defer t.contractsMu.RUnlock()

	var chainIDs []uint64
	for chainID := range t.contracts {
		chainIDs = append(chainIDs, chainID)
	}
	return chainIDs, nil
}

func (t *TokenDirectory) GetAllTokens(ctx context.Context) ([]ContractInfo, error) {
	t.contractsMu.RLock()

	// Get all chain IDs first
	var chainIDs []uint64
	for chainID := range t.contracts {
		chainIDs = append(chainIDs, chainID)
	}

	t.contractsMu.RUnlock()

	// Now fetch tokens for each chain ID using GetTokens which has its own locking
	var tokens []ContractInfo
	for _, chainID := range chainIDs {
		list, err := t.GetTokens(ctx, chainID)
		if err != nil {
			return nil, err
		}
		tokens = append(tokens, list...)
	}

	return tokens, nil
}

func (t *TokenDirectory) GetTokens(ctx context.Context, chainID uint64) ([]ContractInfo, error) {
	t.contractsMu.RLock()
	defer t.contractsMu.RUnlock()

	if _, ok := t.contracts[chainID]; !ok {
		return []ContractInfo{}, nil
	}
	tokens := make([]ContractInfo, 0, len(t.contracts[chainID]))
	for _, token := range t.contracts[chainID] {
		tokens = append(tokens, token)
	}
	return tokens, nil
}

func normalizeTokens(provider Provider, tokenList *TokenList) {
	// normalize addresses
	for i, info := range tokenList.Tokens {
		tokenList.Tokens[i].Address = strings.ToLower(info.Address)
		tokenList.Tokens[i].Extensions.OriginAddress = strings.ToLower(info.Extensions.OriginAddress)
		tokenList.Tokens[i].Type = strings.ToUpper(tokenList.TokenStandard)
		// add the provider verification stamp
		tokenList.Tokens[i].Extensions.Verified = !info.Extensions.Blacklist
		tokenList.Tokens[i].Extensions.VerifiedBy = provider.GetID()
	}
}
