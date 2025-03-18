package tokendirectory

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSequenceProvider(t *testing.T) {
	tests := []struct {
		name      string
		baseURL   string
		client    HTTPClient
		expectErr bool
	}{
		{
			name:      "Valid provider with default client",
			baseURL:   "https://example.com",
			client:    nil,
			expectErr: false,
		},
		{
			name:      "Valid provider with custom client",
			baseURL:   "https://example.com",
			client:    http.DefaultClient,
			expectErr: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			provider, err := NewSequenceProvider(tc.baseURL, tc.client)
			if tc.expectErr {
				assert.Error(t, err)
				assert.Nil(t, provider)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, provider)
				assert.Equal(t, "sequence-token-directory", provider.GetID())
			}
		})
	}
}

func TestSequenceProvider_GetID(t *testing.T) {
	provider, err := NewSequenceProvider("https://example.com", nil)
	require.NoError(t, err)
	assert.Equal(t, "sequence-token-directory", provider.GetID())
}

func TestSequenceProvider_GetConfig(t *testing.T) {
	tests := []struct {
		name             string
		serverResponse   interface{}
		statusCode       int
		expectedChainIDs []uint64
		expectedSources  []SourceType
		expectErr        bool
	}{
		{
			name: "Valid config response",
			serverResponse: struct {
				ChainIds []uint64     `json:"chainIds"`
				Types    []SourceType `json:"sources"`
			}{
				ChainIds: []uint64{1, 137, 43114},
				Types:    []SourceType{SourceTypeERC20, SourceTypeERC721},
			},
			statusCode:       http.StatusOK,
			expectedChainIDs: []uint64{1, 137, 43114},
			expectedSources:  []SourceType{SourceTypeERC20, SourceTypeERC721},
			expectErr:        false,
		},
		{
			name:             "Server error",
			serverResponse:   nil,
			statusCode:       http.StatusInternalServerError,
			expectedChainIDs: nil,
			expectedSources:  nil,
			expectErr:        true,
		},
		{
			name:             "Invalid JSON response",
			serverResponse:   "invalid json",
			statusCode:       http.StatusOK,
			expectedChainIDs: nil,
			expectedSources:  nil,
			expectErr:        true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Setup mock server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Contains(t, r.URL.Path, "/token-directory")
				assert.Equal(t, "GET", r.Method)

				w.WriteHeader(tc.statusCode)
				if tc.statusCode == http.StatusOK {
					if _, ok := tc.serverResponse.(string); ok {
						fmt.Fprintln(w, tc.serverResponse)
					} else {
						if err := json.NewEncoder(w).Encode(tc.serverResponse); err != nil {
							t.Fatalf("Failed to encode response: %v", err)
						}
					}
				}
			}))
			defer server.Close()

			// Create provider with mock server URL
			baseURL := server.URL
			if baseURL[len(baseURL)-1] != '/' {
				baseURL = baseURL + "/"
			}
			provider, err := NewSequenceProvider(baseURL, server.Client())
			require.NoError(t, err)

			// Test GetConfig
			chainIDs, sources, err := provider.GetConfig(context.Background())
			if tc.expectErr {
				assert.Error(t, err)
				assert.Nil(t, chainIDs)
				assert.Nil(t, sources)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expectedChainIDs, chainIDs)
				assert.Equal(t, tc.expectedSources, sources)
			}
		})
	}
}

func TestSequenceProvider_FetchTokenList(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name           string
		chainID        uint64
		source         SourceType
		serverResponse interface{}
		statusCode     int
		expectedResult *TokenList
		expectErr      bool
	}{
		{
			name:    "Valid token list response",
			chainID: 1,
			source:  SourceTypeERC20,
			serverResponse: TokenList{
				Name:      "Ethereum Tokens",
				ChainID:   1,
				Timestamp: &now,
				Tokens: []ContractInfo{
					{
						ChainID:  1,
						Address:  "0x1234567890123456789012345678901234567890",
						Name:     "Test Token",
						Symbol:   "TEST",
						Decimals: 18,
					},
				},
			},
			statusCode: http.StatusOK,
			expectedResult: &TokenList{
				Name:      "Ethereum Tokens",
				ChainID:   1,
				Timestamp: &now,
				Tokens: []ContractInfo{
					{
						ChainID:  1,
						Address:  "0x1234567890123456789012345678901234567890",
						Name:     "Test Token",
						Symbol:   "TEST",
						Decimals: 18,
					},
				},
			},
			expectErr: false,
		},
		{
			name:    "Only tokens array response",
			chainID: 1,
			source:  SourceTypeERC20,
			serverResponse: []ContractInfo{
				{
					ChainID:  1,
					Address:  "0x1234567890123456789012345678901234567890",
					Name:     "Test Token",
					Symbol:   "TEST",
					Decimals: 18,
				},
			},
			statusCode: http.StatusOK,
			expectedResult: &TokenList{
				Name:    "1",
				ChainID: 1,
				Tokens: []ContractInfo{
					{
						ChainID:  1,
						Address:  "0x1234567890123456789012345678901234567890",
						Name:     "Test Token",
						Symbol:   "TEST",
						Decimals: 18,
					},
				},
			},
			expectErr: false,
		},
		{
			name:           "HTTP error",
			chainID:        1,
			source:         SourceTypeERC20,
			serverResponse: nil,
			statusCode:     http.StatusInternalServerError,
			expectedResult: nil,
			expectErr:      true,
		},
		{
			name:           "Invalid JSON response",
			chainID:        1,
			source:         SourceTypeERC20,
			serverResponse: "invalid json",
			statusCode:     http.StatusOK,
			expectedResult: nil,
			expectErr:      true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Setup mock server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Check that path contains the expected components
				assert.Contains(t, r.URL.Path, fmt.Sprintf("token-directory/%d/%s.json", tc.chainID, tc.source))
				assert.Equal(t, "GET", r.Method)

				w.WriteHeader(tc.statusCode)
				if tc.statusCode == http.StatusOK {
					if jsonStr, ok := tc.serverResponse.(string); ok {
						fmt.Fprintln(w, jsonStr)
					} else {
						if err := json.NewEncoder(w).Encode(tc.serverResponse); err != nil {
							t.Fatalf("Failed to encode response: %v", err)
						}
					}
				}
			}))
			defer server.Close()

			// Create provider with mock server URL
			baseURL := server.URL
			if baseURL[len(baseURL)-1] != '/' {
				baseURL = baseURL + "/"
			}
			provider, err := NewSequenceProvider(baseURL, server.Client())
			require.NoError(t, err)

			// Test FetchTokenList
			result, err := provider.FetchTokenList(context.Background(), tc.chainID, tc.source)
			if tc.expectErr {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, tc.expectedResult.Name, result.Name)
				assert.Equal(t, tc.expectedResult.ChainID, result.ChainID)

				// Compare tokens
				assert.Equal(t, len(tc.expectedResult.Tokens), len(result.Tokens))
				for i, expectedToken := range tc.expectedResult.Tokens {
					assert.Equal(t, expectedToken.ChainID, result.Tokens[i].ChainID)
					assert.Equal(t, expectedToken.Address, result.Tokens[i].Address)
					assert.Equal(t, expectedToken.Name, result.Tokens[i].Name)
					assert.Equal(t, expectedToken.Symbol, result.Tokens[i].Symbol)
					assert.Equal(t, expectedToken.Decimals, result.Tokens[i].Decimals)
				}
			}
		})
	}
}

// MockHTTPClient implements the HTTPClient interface for testing
type MockHTTPClient struct {
	DoFunc func(req *http.Request) (*http.Response, error)
}

func (m *MockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	return m.DoFunc(req)
}

func TestSequenceProvider_FetchTokenList_ClientError(t *testing.T) {
	// Create a mock client that returns an error
	mockClient := &MockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			return nil, fmt.Errorf("network error")
		},
	}

	provider, err := NewSequenceProvider("https://example.com", mockClient)
	require.NoError(t, err)

	// Test FetchTokenList with client error
	result, err := provider.FetchTokenList(context.Background(), 1, SourceTypeERC20)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "fetching: network error")
}
