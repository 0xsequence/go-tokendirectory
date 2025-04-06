package tokendirectory

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTokenListProvider(t *testing.T) {
	tests := []struct {
		name            string
		sources         []map[uint64]map[SourceType]string
		client          *http.Client
		expectDefaultID string
	}{
		{
			name:            "Default sources",
			sources:         nil,
			client:          nil,
			expectDefaultID: "tokenlist-directory",
		},
		{
			name: "Custom sources",
			sources: []map[uint64]map[SourceType]string{
				{
					1: {
						SourceTypeERC20: "https://example.com/tokens",
					},
				},
			},
			client:          nil,
			expectDefaultID: "tokenlist-directory",
		},
		{
			name:            "Default sources with custom client",
			sources:         nil,
			client:          &http.Client{},
			expectDefaultID: "tokenlist-directory",
		},
		{
			name: "Custom sources with custom client",
			sources: []map[uint64]map[SourceType]string{
				{
					1: {
						SourceTypeERC20: "https://example.com/tokens",
					},
				},
			},
			client:          &http.Client{},
			expectDefaultID: "tokenlist-directory",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var provider Provider
			if tc.sources == nil && tc.client == nil {
				provider = NewTokenListProvider(nil)
			} else if tc.sources == nil && tc.client != nil {
				provider = NewTokenListProvider(nil, tc.client)
			} else if tc.sources != nil && tc.client == nil {
				provider = NewTokenListProvider(tc.sources)
			} else {
				provider = NewTokenListProvider(tc.sources, tc.client)
			}

			assert.NotNil(t, provider)
			assert.Equal(t, tc.expectDefaultID, provider.GetID())

			// Verify it's the correct type
			_, ok := provider.(tokenListProvider)
			assert.True(t, ok)
		})
	}
}

func TestTokenListProvider_GetID(t *testing.T) {
	provider := tokenListProvider{
		id: "test-provider",
	}

	assert.Equal(t, "test-provider", provider.GetID())
}

func TestTokenListProvider_GetConfig(t *testing.T) {
	// Test with empty sources
	t.Run("Empty sources", func(t *testing.T) {
		provider := tokenListProvider{
			sources: map[uint64]map[SourceType]string{},
		}

		chainIDs, sources, err := provider.GetConfig(context.Background())
		require.NoError(t, err)
		assert.Empty(t, chainIDs)
		assert.Equal(t, []SourceType{SourceTypeERC20, SourceTypeERC721, SourceTypeERC1155}, sources)
	})

	// Test with multiple chains and sources
	t.Run("Multiple chains and sources", func(t *testing.T) {
		provider := tokenListProvider{
			sources: map[uint64]map[SourceType]string{
				1: {
					SourceTypeERC20:  "url1",
					SourceTypeERC721: "url2",
				},
				10: {
					SourceTypeERC1155: "url3",
				},
			},
		}

		chainIDs, sources, err := provider.GetConfig(context.Background())
		require.NoError(t, err)
		assert.ElementsMatch(t, []uint64{1, 10}, chainIDs)
		assert.Equal(t, []SourceType{SourceTypeERC20, SourceTypeERC721, SourceTypeERC1155}, sources)
	})
}

func TestTokenListProvider_FetchTokenList(t *testing.T) {
	// Test missing chain
	t.Run("Missing chain", func(t *testing.T) {
		provider := tokenListProvider{
			sources: map[uint64]map[SourceType]string{},
		}

		list, err := provider.FetchTokenList(context.Background(), 1, SourceTypeERC20)
		assert.Error(t, err)
		assert.Nil(t, list)
		assert.Contains(t, err.Error(), "no sources for chain 1")
	})

	// Test missing source type
	t.Run("Missing source type", func(t *testing.T) {
		provider := tokenListProvider{
			sources: map[uint64]map[SourceType]string{
				1: {
					SourceTypeERC721: "url1",
				},
			},
		}

		list, err := provider.FetchTokenList(context.Background(), 1, SourceTypeERC20)
		assert.Nil(t, err)
		assert.Nil(t, list)
	})

	// Test successful standard tokenlist
	t.Run("Standard token list format", func(t *testing.T) {
		mockTime := time.Now()
		mockTokenList := TokenList{
			Name:          "Test List",
			ChainID:       1,
			TokenStandard: "ERC20",
			Timestamp:     &mockTime,
			Tokens: []ContractInfo{
				{
					ChainID:  1,
					Address:  "0x1234567890123456789012345678901234567890",
					Name:     "Test Token",
					Symbol:   "TST",
					Decimals: PtrTo(uint64(18)),
				},
			},
		}

		tokenListJSON, err := json.Marshal(mockTokenList)
		require.NoError(t, err)

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			if _, err := w.Write(tokenListJSON); err != nil {
				t.Fatalf("Failed to write response: %v", err)
			}
		}))
		defer server.Close()

		// Create test provider using the updated constructor
		sources := []map[uint64]map[SourceType]string{
			{
				1: {
					SourceTypeERC20: server.URL,
				},
			},
		}
		provider := NewTokenListProvider(sources)

		list, err := provider.FetchTokenList(context.Background(), 1, SourceTypeERC20)
		require.NoError(t, err)
		assert.NotNil(t, list)
		assert.Equal(t, "Test List", list.Name)
		assert.Equal(t, uint64(1), list.ChainID)
		assert.Equal(t, "ERC20", list.TokenStandard)
		assert.Equal(t, 1, len(list.Tokens))
		assert.Equal(t, "Test Token", list.Tokens[0].Name)
	})

	// Test non-standard tokenlist (just tokens array)
	t.Run("Non-standard token list format (just tokens array)", func(t *testing.T) {
		tokens := []ContractInfo{
			{
				ChainID:  1,
				Address:  "0x1234567890123456789012345678901234567890",
				Name:     "Test Token",
				Symbol:   "TST",
				Decimals: PtrTo(uint64(18)),
			},
		}

		tokensJSON, err := json.Marshal(tokens)
		require.NoError(t, err)

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			if _, err := w.Write(tokensJSON); err != nil {
				t.Fatalf("Failed to write response: %v", err)
			}
		}))
		defer server.Close()

		// Create test provider using the updated constructor
		sources := []map[uint64]map[SourceType]string{
			{
				1: {
					SourceTypeERC20: server.URL,
				},
			},
		}
		provider := NewTokenListProvider(sources)

		list, err := provider.FetchTokenList(context.Background(), 1, SourceTypeERC20)
		require.NoError(t, err)
		assert.NotNil(t, list)
		assert.Equal(t, "1", list.Name) // Name is string of chainID
		assert.Equal(t, uint64(1), list.ChainID)
		assert.Equal(t, 1, len(list.Tokens))
		assert.Equal(t, "Test Token", list.Tokens[0].Name)
	})

	// Test HTTP error
	t.Run("HTTP error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		// Create test provider using the updated constructor
		sources := []map[uint64]map[SourceType]string{
			{
				1: {
					SourceTypeERC20: server.URL,
				},
			},
		}
		provider := NewTokenListProvider(sources)

		list, err := provider.FetchTokenList(context.Background(), 1, SourceTypeERC20)
		assert.Error(t, err)
		assert.Nil(t, list)
		assert.Contains(t, err.Error(), "fetching: 500")
	})

	// Test invalid JSON response
	t.Run("Invalid JSON response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			if _, err := w.Write([]byte("invalid json")); err != nil {
				t.Fatalf("Failed to write response: %v", err)
			}
		}))
		defer server.Close()

		// Create test provider using the updated constructor
		sources := []map[uint64]map[SourceType]string{
			{
				1: {
					SourceTypeERC20: server.URL,
				},
			},
		}
		provider := NewTokenListProvider(sources)

		list, err := provider.FetchTokenList(context.Background(), 1, SourceTypeERC20)
		assert.Error(t, err)
		assert.Nil(t, list)
		assert.Contains(t, err.Error(), "decoding json")
	})

	// Test context cancellation
	t.Run("Context cancellation", func(t *testing.T) {
		// Create test provider using the updated constructor
		sources := []map[uint64]map[SourceType]string{
			{
				1: {
					SourceTypeERC20: "https://example.com",
				},
			},
		}
		provider := NewTokenListProvider(sources)

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel the context before the request

		list, err := provider.FetchTokenList(ctx, 1, SourceTypeERC20)
		assert.Error(t, err)
		assert.Nil(t, list)
		assert.Contains(t, err.Error(), "context canceled")
	})

	// Test with custom HTTP client
	t.Run("Custom HTTP client", func(t *testing.T) {
		mockTime := time.Now()
		mockTokenList := TokenList{
			Name:          "Test List",
			ChainID:       1,
			TokenStandard: "ERC20",
			Timestamp:     &mockTime,
			Tokens: []ContractInfo{
				{
					ChainID:  1,
					Address:  "0x1234567890123456789012345678901234567890",
					Name:     "Test Token",
					Symbol:   "TST",
					Decimals: PtrTo(uint64(18)),
				},
			},
		}

		tokenListJSON, err := json.Marshal(mockTokenList)
		require.NoError(t, err)

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			if _, err := w.Write(tokenListJSON); err != nil {
				t.Fatalf("Failed to write response: %v", err)
			}
		}))
		defer server.Close()

		// Create a custom HTTP client
		customClient := &http.Client{
			Timeout: 5 * time.Second,
		}

		// Create test provider using the updated constructor with a custom client
		sources := []map[uint64]map[SourceType]string{
			{
				1: {
					SourceTypeERC20: server.URL,
				},
			},
		}
		provider := NewTokenListProvider(sources, customClient)

		list, err := provider.FetchTokenList(context.Background(), 1, SourceTypeERC20)
		require.NoError(t, err)
		assert.NotNil(t, list)
		assert.Equal(t, "Test List", list.Name)
		assert.Equal(t, uint64(1), list.ChainID)
		assert.Equal(t, 1, len(list.Tokens))
		assert.Equal(t, "Test Token", list.Tokens[0].Name)
	})
}

func PtrTo[T any](v T) *T {
	return &v
}
