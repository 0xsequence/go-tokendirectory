package tokendirectory

import (
	"context"
	"io"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/0xsequence/go-sequence/lib/prototyp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockProvider implements the Provider interface for testing
type MockProvider struct {
	ID           string
	ChainIDs     []uint64
	Sources      []SourceType
	TokenLists   map[uint64]map[SourceType]*TokenList
	ConfigErr    error
	FetchListErr error
	FetchCallsMu sync.Mutex
	FetchCalls   map[string]int
	ConfigCalls  int
}

func NewMockProvider(id string, chainIDs []uint64, sources []SourceType) *MockProvider {
	return &MockProvider{
		ID:         id,
		ChainIDs:   chainIDs,
		Sources:    sources,
		TokenLists: make(map[uint64]map[SourceType]*TokenList),
		FetchCalls: make(map[string]int),
	}
}

func (m *MockProvider) GetID() string {
	return m.ID
}

func (m *MockProvider) GetConfig(ctx context.Context) ([]uint64, []SourceType, error) {
	m.ConfigCalls++
	if m.ConfigErr != nil {
		return nil, nil, m.ConfigErr
	}
	return m.ChainIDs, m.Sources, nil
}

func (m *MockProvider) FetchTokenList(ctx context.Context, chainID uint64, source SourceType) (*TokenList, error) {
	key := generateKey(chainID, source)
	m.FetchCallsMu.Lock()
	m.FetchCalls[key]++
	m.FetchCallsMu.Unlock()

	if m.FetchListErr != nil {
		return nil, m.FetchListErr
	}

	chainLists, ok := m.TokenLists[chainID]
	if !ok {
		return nil, nil
	}

	list, ok := chainLists[source]
	if !ok {
		return nil, nil
	}

	return list, nil
}

func (m *MockProvider) AddTokenList(chainID uint64, source SourceType, list *TokenList) {
	if _, ok := m.TokenLists[chainID]; !ok {
		m.TokenLists[chainID] = make(map[SourceType]*TokenList)
	}
	m.TokenLists[chainID][source] = list
}

func generateKey(chainID uint64, source SourceType) string {
	return string(source) + "-" + string(rune(chainID))
}

func TestNewTokenDirectory(t *testing.T) {
	// Test with default options
	t.Run("Default Options", func(t *testing.T) {
		dir, err := NewTokenDirectory()
		require.NoError(t, err)
		assert.NotNil(t, dir)

		// Check default values
		assert.Equal(t, time.Minute*15, dir.updateInterval)
		assert.Len(t, dir.providers, 1)
		assert.NotNil(t, dir.log)
	})

	// Test with custom providers
	t.Run("Custom Provider", func(t *testing.T) {
		mockProvider := NewMockProvider("mock-provider", []uint64{1}, []SourceType{SourceTypeERC20})

		dir, err := NewTokenDirectory(WithProviders(mockProvider))
		require.NoError(t, err)
		assert.NotNil(t, dir)
		assert.Len(t, dir.providers, 1)
		assert.Contains(t, dir.providers, "mock-provider")
	})

	// Test with custom logger
	t.Run("Custom Logger", func(t *testing.T) {
		logger := slog.New(slog.NewTextHandler(io.Discard, nil))

		dir, err := NewTokenDirectory(WithLogger(logger))
		require.NoError(t, err)
		assert.NotNil(t, dir)
		assert.Equal(t, logger, dir.log)
	})

	// Test with custom update interval
	t.Run("Custom Update Interval", func(t *testing.T) {
		dir, err := NewTokenDirectory(WithUpdateInterval(time.Hour))
		require.NoError(t, err)
		assert.NotNil(t, dir)
		assert.Equal(t, time.Hour, dir.updateInterval)
	})

	// Test with invalid update interval
	t.Run("Invalid Update Interval", func(t *testing.T) {
		dir, err := NewTokenDirectory(WithUpdateInterval(time.Second * 30))
		assert.Error(t, err)
		assert.Nil(t, dir)
	})

	// Test with chain IDs filter
	t.Run("Chain IDs Filter", func(t *testing.T) {
		chainIDs := []uint64{1, 137}
		dir, err := NewTokenDirectory(WithChainIDs(chainIDs...))
		require.NoError(t, err)
		assert.NotNil(t, dir)
		assert.Equal(t, chainIDs, dir.chainIDs)
	})

	// Test with sources filter
	t.Run("Sources Filter", func(t *testing.T) {
		sources := []SourceType{SourceTypeERC20, SourceTypeERC721}
		dir, err := NewTokenDirectory(WithSources(sources...))
		require.NoError(t, err)
		assert.NotNil(t, dir)
		assert.Equal(t, sources, dir.sources)
	})

	// Test with multiple options
	t.Run("Multiple Options", func(t *testing.T) {
		mockProvider := NewMockProvider("mock-provider", []uint64{1}, []SourceType{SourceTypeERC20})
		logger := slog.New(slog.NewTextHandler(io.Discard, nil))

		dir, err := NewTokenDirectory(
			WithProviders(mockProvider),
			WithLogger(logger),
			WithUpdateInterval(time.Hour),
			WithChainIDs(1, 137),
			WithSources(SourceTypeERC20),
		)
		require.NoError(t, err)
		assert.NotNil(t, dir)
		assert.Len(t, dir.providers, 1)
		assert.Equal(t, logger, dir.log)
		assert.Equal(t, time.Hour, dir.updateInterval)
		assert.Equal(t, []uint64{1, 137}, dir.chainIDs)
		assert.Equal(t, []SourceType{SourceTypeERC20}, dir.sources)
	})
}

func TestTokenDirectory_updateSources(t *testing.T) {
	// Create mock provider with sample data
	mockProvider := NewMockProvider("mock-provider", []uint64{1, 137}, []SourceType{SourceTypeERC20, SourceTypeERC721})

	// Add token lists to the mock provider
	mockProvider.AddTokenList(1, SourceTypeERC20, &TokenList{
		Name:          "Ethereum ERC20",
		ChainID:       1,
		TokenStandard: "ERC20",
		Tokens: []ContractInfo{
			{
				ChainID:  1,
				Address:  "0x1234567890123456789012345678901234567890",
				Name:     "Token 1",
				Symbol:   "TKN1",
				Decimals: 18,
			},
		},
	})

	mockProvider.AddTokenList(137, SourceTypeERC20, &TokenList{
		Name:          "Polygon ERC20",
		ChainID:       137,
		TokenStandard: "ERC20",
		Tokens: []ContractInfo{
			{
				ChainID:  137,
				Address:  "0xabcdef0123456789abcdef0123456789abcdef01",
				Name:     "Token 2",
				Symbol:   "TKN2",
				Decimals: 18,
			},
		},
	})

	// Create token directory with mock provider
	dir, err := NewTokenDirectory(WithProviders(mockProvider))
	require.NoError(t, err)

	// Update sources and verify the data is populated
	err = dir.updateSources(context.Background())
	require.NoError(t, err)

	// Check that the provider was called correctly
	assert.Equal(t, 1, mockProvider.ConfigCalls)
	// The provider should be called for each chain and source type combination
	// (2 chains x 2 source types = 4 calls)
	assert.Equal(t, 4, len(mockProvider.FetchCalls))

	// Verify contract info is stored correctly
	assert.Len(t, dir.contracts, 2)
	assert.Len(t, dir.contracts[1], 1)
	assert.Len(t, dir.contracts[137], 1)

	// Test with chain ID filter
	t.Run("With Chain ID Filter", func(t *testing.T) {
		mockProvider := NewMockProvider("mock-provider", []uint64{1, 137}, []SourceType{SourceTypeERC20})
		mockProvider.AddTokenList(1, SourceTypeERC20, &TokenList{
			Name:          "Ethereum ERC20",
			ChainID:       1,
			TokenStandard: "ERC20",
			Tokens: []ContractInfo{
				{
					ChainID:  1,
					Address:  "0x1234567890123456789012345678901234567890",
					Name:     "Token 1",
					Symbol:   "TKN1",
					Decimals: 18,
				},
			},
		})
		mockProvider.AddTokenList(137, SourceTypeERC20, &TokenList{
			Name:          "Polygon ERC20",
			ChainID:       137,
			TokenStandard: "ERC20",
			Tokens: []ContractInfo{
				{
					ChainID:  137,
					Address:  "0xabcdef0123456789abcdef0123456789abcdef01",
					Name:     "Token 2",
					Symbol:   "TKN2",
					Decimals: 18,
				},
			},
		})

		dir, err := NewTokenDirectory(
			WithProviders(mockProvider),
			WithChainIDs(1), // Only fetch Ethereum
		)
		require.NoError(t, err)

		err = dir.updateSources(context.Background())
		require.NoError(t, err)

		// Verify only Ethereum contracts are stored
		assert.Len(t, dir.contracts, 1)
		assert.Len(t, dir.contracts[1], 1)
		assert.NotContains(t, dir.contracts, uint64(137))
	})

	// Test with source filter
	t.Run("With Source Filter", func(t *testing.T) {
		mockProvider := NewMockProvider("mock-provider", []uint64{1}, []SourceType{SourceTypeERC20, SourceTypeERC721})
		mockProvider.AddTokenList(1, SourceTypeERC20, &TokenList{
			Name:          "Ethereum ERC20",
			ChainID:       1,
			TokenStandard: "ERC20",
			Tokens: []ContractInfo{
				{
					ChainID:  1,
					Address:  "0x1234567890123456789012345678901234567890",
					Name:     "Token 1",
					Symbol:   "TKN1",
					Decimals: 18,
				},
			},
		})
		mockProvider.AddTokenList(1, SourceTypeERC721, &TokenList{
			Name:          "Ethereum ERC721",
			ChainID:       1,
			TokenStandard: "ERC721",
			Tokens: []ContractInfo{
				{
					ChainID:  1,
					Address:  "0xabcdef0123456789abcdef0123456789abcdef01",
					Name:     "NFT 1",
					Symbol:   "NFT1",
					Decimals: 0,
				},
			},
		})

		dir, err := NewTokenDirectory(
			WithProviders(mockProvider),
			WithSources(SourceTypeERC20), // Only fetch ERC20
		)
		require.NoError(t, err)

		err = dir.updateSources(context.Background())
		require.NoError(t, err)

		// Verify only ERC20 tokens are stored
		tokens, err := dir.GetTokens(context.Background(), 1)
		require.NoError(t, err)
		assert.Len(t, tokens, 1)
		assert.Equal(t, "Token 1", tokens[0].Name)
	})
}

func TestTokenDirectory_GetContractInfo(t *testing.T) {
	// Create mock provider with sample data
	mockProvider := NewMockProvider("mock-provider", []uint64{1}, []SourceType{SourceTypeERC20})

	// Add token list to the mock provider
	tokenAddress := "0x1234567890123456789012345678901234567890"
	mockProvider.AddTokenList(1, SourceTypeERC20, &TokenList{
		Name:          "Ethereum ERC20",
		ChainID:       1,
		TokenStandard: "ERC20",
		Tokens: []ContractInfo{
			{
				ChainID:  1,
				Address:  tokenAddress,
				Name:     "Token 1",
				Symbol:   "TKN1",
				Decimals: 18,
			},
		},
	})

	// Create token directory with mock provider
	dir, err := NewTokenDirectory(WithProviders(mockProvider))
	require.NoError(t, err)

	// Update sources to populate data
	err = dir.updateSources(context.Background())
	require.NoError(t, err)

	// Test getting contract info for existing contract
	info, exists, err := dir.GetContractInfo(context.Background(), 1, prototyp.HashFromString(tokenAddress))
	require.NoError(t, err)
	assert.True(t, exists)
	assert.Equal(t, "Token 1", info.Name)
	assert.Equal(t, "TKN1", info.Symbol)
	assert.Equal(t, uint64(18), info.Decimals)

	// Test getting contract info for non-existent contract
	info, exists, err = dir.GetContractInfo(context.Background(), 1, prototyp.HashFromString("0xdead"))
	require.Error(t, err)
	assert.False(t, exists)
	assert.Empty(t, info)

	// Test getting contract info for non-existent chain
	info, exists, err = dir.GetContractInfo(context.Background(), 999, prototyp.HashFromString(tokenAddress))
	require.Error(t, err)
	assert.False(t, exists)
	assert.Empty(t, info)
}

func TestTokenDirectory_GetNetworks(t *testing.T) {
	// Create mock provider with multiple networks
	mockProvider := NewMockProvider("mock-provider", []uint64{1, 137}, []SourceType{SourceTypeERC20})

	// Add token lists to the mock provider
	mockProvider.AddTokenList(1, SourceTypeERC20, &TokenList{
		Name:          "Ethereum ERC20",
		ChainID:       1,
		TokenStandard: "ERC20",
		Tokens: []ContractInfo{
			{
				ChainID:  1,
				Address:  "0x1234567890123456789012345678901234567890",
				Name:     "Token 1",
				Symbol:   "TKN1",
				Decimals: 18,
			},
		},
	})

	mockProvider.AddTokenList(137, SourceTypeERC20, &TokenList{
		Name:          "Polygon ERC20",
		ChainID:       137,
		TokenStandard: "ERC20",
		Tokens: []ContractInfo{
			{
				ChainID:  137,
				Address:  "0xabcdef0123456789abcdef0123456789abcdef01",
				Name:     "Token 2",
				Symbol:   "TKN2",
				Decimals: 18,
			},
		},
	})

	// Create token directory with mock provider
	dir, err := NewTokenDirectory(WithProviders(mockProvider))
	require.NoError(t, err)

	// Update sources to populate data
	err = dir.updateSources(context.Background())
	require.NoError(t, err)

	// Test getting networks
	networks, err := dir.GetNetworks(context.Background())
	require.NoError(t, err)
	assert.Len(t, networks, 2)
	assert.Contains(t, networks, uint64(1))
	assert.Contains(t, networks, uint64(137))
}

func TestTokenDirectory_GetTokens(t *testing.T) {
	// Create mock provider with sample data
	mockProvider := NewMockProvider("mock-provider", []uint64{1}, []SourceType{SourceTypeERC20, SourceTypeERC721})

	// Add token lists to the mock provider
	mockProvider.AddTokenList(1, SourceTypeERC20, &TokenList{
		Name:          "Ethereum ERC20",
		ChainID:       1,
		TokenStandard: "ERC20",
		Tokens: []ContractInfo{
			{
				ChainID:  1,
				Address:  "0x1234567890123456789012345678901234567890",
				Name:     "Token 1",
				Symbol:   "TKN1",
				Decimals: 18,
			},
		},
	})

	mockProvider.AddTokenList(1, SourceTypeERC721, &TokenList{
		Name:          "Ethereum ERC721",
		ChainID:       1,
		TokenStandard: "ERC721",
		Tokens: []ContractInfo{
			{
				ChainID:  1,
				Address:  "0xabcdef0123456789abcdef0123456789abcdef01",
				Name:     "NFT 1",
				Symbol:   "NFT1",
				Decimals: 0,
			},
		},
	})

	// Create token directory with mock provider
	dir, err := NewTokenDirectory(WithProviders(mockProvider))
	require.NoError(t, err)

	// Update sources to populate data
	err = dir.updateSources(context.Background())
	require.NoError(t, err)

	// Test getting all tokens for a chain
	tokens, err := dir.GetTokens(context.Background(), 1)
	require.NoError(t, err)
	assert.Len(t, tokens, 2)

	// Verify token names
	tokenNames := []string{tokens[0].Name, tokens[1].Name}
	assert.Contains(t, tokenNames, "Token 1")
	assert.Contains(t, tokenNames, "NFT 1")
}

func TestTokenDirectory_GetAllTokens(t *testing.T) {
	// Create mock provider with multiple networks
	mockProvider := NewMockProvider("mock-provider", []uint64{1, 137}, []SourceType{SourceTypeERC20})

	// Add token lists to the mock provider
	mockProvider.AddTokenList(1, SourceTypeERC20, &TokenList{
		Name:          "Ethereum ERC20",
		ChainID:       1,
		TokenStandard: "ERC20",
		Tokens: []ContractInfo{
			{
				ChainID:  1,
				Address:  "0x1234567890123456789012345678901234567890",
				Name:     "Token 1",
				Symbol:   "TKN1",
				Decimals: 18,
			},
		},
	})

	mockProvider.AddTokenList(137, SourceTypeERC20, &TokenList{
		Name:          "Polygon ERC20",
		ChainID:       137,
		TokenStandard: "ERC20",
		Tokens: []ContractInfo{
			{
				ChainID:  137,
				Address:  "0xabcdef0123456789abcdef0123456789abcdef01",
				Name:     "Token 2",
				Symbol:   "TKN2",
				Decimals: 18,
			},
		},
	})

	// Create token directory with mock provider
	dir, err := NewTokenDirectory(WithProviders(mockProvider))
	require.NoError(t, err)

	// Update sources to populate data
	err = dir.updateSources(context.Background())
	require.NoError(t, err)

	// Test getting all tokens across all chains
	allTokens, err := dir.GetAllTokens(context.Background())
	require.NoError(t, err)
	assert.Len(t, allTokens, 2)

	// Verify tokens from both chains are included
	chainIDs := []uint64{allTokens[0].ChainID, allTokens[1].ChainID}
	assert.Contains(t, chainIDs, uint64(1))
	assert.Contains(t, chainIDs, uint64(137))
}

func TestTokenDirectory_OnUpdate(t *testing.T) {
	// Create a channel to track updates
	updateCh := make(chan []ContractInfo, 10)

	// Create update function
	onUpdate := func(ctx context.Context, chainID uint64, contractInfoList []ContractInfo) {
		updateCh <- contractInfoList
	}

	// Create mock provider with sample data
	mockProvider := NewMockProvider("mock-provider", []uint64{1}, []SourceType{SourceTypeERC20})

	// Add token list to the mock provider
	mockProvider.AddTokenList(1, SourceTypeERC20, &TokenList{
		Name:          "Ethereum ERC20",
		ChainID:       1,
		TokenStandard: "ERC20",
		Tokens: []ContractInfo{
			{
				ChainID:  1,
				Address:  "0x1234567890123456789012345678901234567890",
				Name:     "Token 1",
				Symbol:   "TKN1",
				Decimals: 18,
			},
		},
	})

	// Create token directory with mock provider and update function
	dir, err := NewTokenDirectory(
		WithProviders(mockProvider),
		WithUpdateFuncs(onUpdate),
	)
	require.NoError(t, err)

	// Update sources to trigger update callback
	err = dir.updateSources(context.Background())
	require.NoError(t, err)

	// Wait for update to be processed
	select {
	case contractInfo := <-updateCh:
		assert.Len(t, contractInfo, 1)
		assert.Equal(t, "Token 1", contractInfo[0].Name)
	case <-time.After(time.Second):
		t.Fatal("Timeout waiting for update callback")
	}
}

func TestTokenDirectory_Run(t *testing.T) {
	// Create mock provider with sample data
	mockProvider := NewMockProvider("mock-provider", []uint64{1}, []SourceType{SourceTypeERC20})

	// Add token list to the mock provider
	mockProvider.AddTokenList(1, SourceTypeERC20, &TokenList{
		Name:          "Ethereum ERC20",
		ChainID:       1,
		TokenStandard: "ERC20",
		Tokens: []ContractInfo{
			{
				ChainID:  1,
				Address:  "0x1234567890123456789012345678901234567890",
				Name:     "Token 1",
				Symbol:   "TKN1",
				Decimals: 18,
			},
		},
	})

	// Create token directory with mock provider and a short update interval
	dir, err := NewTokenDirectory(
		WithProviders(mockProvider),
		WithUpdateInterval(time.Minute*2), // Set a long enough interval to not trigger during test
	)
	require.NoError(t, err)

	// Start the directory in a goroutine
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		err := dir.Run(ctx)
		require.NoError(t, err)
	}()

	// Wait for it to start
	time.Sleep(time.Millisecond * 100)

	// Verify it's running
	assert.True(t, dir.IsRunning())

	// Verify initial fetch happened by using the public API methods instead of
	// directly accessing the contracts map
	networks, err := dir.GetNetworks(ctx)
	require.NoError(t, err)
	assert.Len(t, networks, 1)
	assert.Contains(t, networks, uint64(1))

	// Check tokens for chain ID 1
	tokens, err := dir.GetTokens(ctx, 1)
	require.NoError(t, err)
	assert.Len(t, tokens, 1)

	// Stop the directory
	dir.Stop()

	// Give it time to stop
	time.Sleep(time.Millisecond * 100)

	// Verify it's stopped
	assert.False(t, dir.IsRunning())
}

func TestTokenDirectory_normalizeTokens(t *testing.T) {
	provider := NewMockProvider("test-provider", nil, nil)

	// Create a token list with mixed case addresses
	tokenList := &TokenList{
		Name:          "Test List",
		ChainID:       1,
		TokenStandard: "erc20",
		Tokens: []ContractInfo{
			{
				ChainID:  1,
				Address:  "0xAbCdEf0123456789AbCdEf0123456789AbCdEf01",
				Name:     "Token 1",
				Symbol:   "TKN1",
				Decimals: 18,
				Extensions: ContractInfoExtension{
					OriginAddress: "0xDeF0123456789AbCdEf0123456789AbCdEf0123",
					Blacklist:     true,
				},
			},
		},
	}

	// Normalize the tokens
	normalizeTokens(provider, tokenList)

	// Verify the addresses are lowercased
	assert.Equal(t, "0xabcdef0123456789abcdef0123456789abcdef01", tokenList.Tokens[0].Address)
	assert.Equal(t, "0xdef0123456789abcdef0123456789abcdef0123", tokenList.Tokens[0].Extensions.OriginAddress)

	// Verify token standard is uppercased
	assert.Equal(t, "ERC20", tokenList.Tokens[0].Type)

	// Verify verification status
	assert.False(t, tokenList.Tokens[0].Extensions.Verified) // Opposite of blacklist
	assert.Equal(t, "test-provider", tokenList.Tokens[0].Extensions.VerifiedBy)
}
