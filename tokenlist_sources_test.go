package tokendirectory

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSourceTypeString(t *testing.T) {
	tests := []struct {
		name     string
		source   SourceType
		expected string
	}{
		{"ERC20", SourceTypeERC20, "erc20"},
		{"ERC721", SourceTypeERC721, "erc721"},
		{"ERC1155", SourceTypeERC1155, "erc1155"},
		{"Misc", SourceTypeMisc, "misc"},
		{"Uniswap", SourceTypeUniswap, "uniswap"},
		{"Sushi", SourceTypeSushi, "sushiswap"},
		{"Pancake", SourceTypePancake, "pancakeswap"},
		{"CoinGecko", SourceTypeCoinGecko, "coingecko"},
		{"Custom", SourceType("custom"), "custom"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.source.String()
			assert.Equal(t, tc.expected, result, "Source type string representation should match")
		})
	}
}

func TestMergeSources(t *testing.T) {
	// Create test source maps
	source1 := map[uint64]map[SourceType]string{
		1: {
			SourceTypeERC20: "source1-erc20",
		},
		2: {
			SourceTypeERC721: "source1-erc721",
		},
	}

	source2 := map[uint64]map[SourceType]string{
		1: {
			SourceTypeERC721: "source2-erc721",
		},
		3: {
			SourceTypeERC1155: "source2-erc1155",
		},
	}

	source3 := map[uint64]map[SourceType]string{
		1: {
			SourceTypeMisc: "source3-misc",
		},
	}

	// Expected merged result
	expected := map[uint64]map[SourceType]string{
		1: {
			SourceTypeERC20:  "source1-erc20",
			SourceTypeERC721: "source2-erc721",
			SourceTypeMisc:   "source3-misc",
		},
		2: {
			SourceTypeERC721: "source1-erc721",
		},
		3: {
			SourceTypeERC1155: "source2-erc1155",
		},
	}

	// Test merging sources
	result := MergeSources(source1, source2, source3)

	// Basic structure check
	assert.Equal(t, expected, result, "MergeSources result should match expected output")

	// Detailed verification of each source map's values
	// Verify source1 values
	for chainID, sourcesMap := range source1 {
		for sourceType, expectedURL := range sourcesMap {
			// Skip if this URL should be overridden by a later source
			if chainID == 1 && sourceType == SourceTypeERC721 {
				continue // This gets overridden by source2
			}

			actualURL, ok := result[chainID][sourceType]
			assert.True(t, ok, "Result should contain %s for chain ID %d from source1", sourceType, chainID)
			assert.Equal(t, expectedURL, actualURL, "URL for %s on chain ID %d from source1 should match", sourceType, chainID)
		}
	}

	// Verify source2 values
	for chainID, sourcesMap := range source2 {
		for sourceType, expectedURL := range sourcesMap {
			actualURL, ok := result[chainID][sourceType]
			assert.True(t, ok, "Result should contain %s for chain ID %d from source2", sourceType, chainID)
			assert.Equal(t, expectedURL, actualURL, "URL for %s on chain ID %d from source2 should match", sourceType, chainID)
		}
	}

	// Verify source3 values
	for chainID, sourcesMap := range source3 {
		for sourceType, expectedURL := range sourcesMap {
			actualURL, ok := result[chainID][sourceType]
			assert.True(t, ok, "Result should contain %s for chain ID %d from source3", sourceType, chainID)
			assert.Equal(t, expectedURL, actualURL, "URL for %s on chain ID %d from source3 should match", sourceType, chainID)
		}
	}

	// Check specific values for clarity
	assert.Equal(t, "source1-erc20", result[1][SourceTypeERC20], "ERC20 value for chain ID 1 should come from source1")
	assert.Equal(t, "source2-erc721", result[1][SourceTypeERC721], "ERC721 value for chain ID 1 should come from source2 (overriding source1)")
	assert.Equal(t, "source3-misc", result[1][SourceTypeMisc], "Misc value for chain ID 1 should come from source3")
	assert.Equal(t, "source1-erc721", result[2][SourceTypeERC721], "ERC721 value for chain ID 2 should come from source1")
	assert.Equal(t, "source2-erc1155", result[3][SourceTypeERC1155], "ERC1155 value for chain ID 3 should come from source2")

	// Test empty sources
	emptyResult := MergeSources()
	assert.Empty(t, emptyResult, "MergeSources with no arguments should return empty map")
}

func TestMergeAllSources(t *testing.T) {
	// Test that merging all predefined sources works without errors
	mergedSources := MergeSources(
		SequenceGithubSources,
		UniswapSources,
		SushiSources,
		CoinGeckoSources,
		PancakeSources,
	)

	// Verify that the merged sources contain all chain IDs from individual sources
	allChainIDs := make(map[uint64]bool)

	// Collect all chain IDs from individual sources
	for chainID := range SequenceGithubSources {
		allChainIDs[chainID] = true
	}
	for chainID := range UniswapSources {
		allChainIDs[chainID] = true
	}
	for chainID := range SushiSources {
		allChainIDs[chainID] = true
	}
	for chainID := range CoinGeckoSources {
		allChainIDs[chainID] = true
	}
	for chainID := range PancakeSources {
		allChainIDs[chainID] = true
	}

	// Verify all chain IDs are present in merged result
	for chainID := range allChainIDs {
		_, ok := mergedSources[chainID]
		assert.True(t, ok, "Chain ID %d should exist in merged sources", chainID)
	}

	// Verify that all URLs from each individual source map are preserved in the merged result

	// Check SequenceGithubSources
	for chainID, sourcesMap := range SequenceGithubSources {
		for sourceType, expectedURL := range sourcesMap {
			actualURL, ok := mergedSources[chainID][sourceType]
			assert.True(t, ok, "Merged sources should contain %s for chain ID %d", sourceType, chainID)
			assert.Equal(t, expectedURL, actualURL, "URL for %s on chain ID %d should match", sourceType, chainID)
		}
	}

	// Check UniswapSources
	for chainID, sourcesMap := range UniswapSources {
		for sourceType, expectedURL := range sourcesMap {
			actualURL, ok := mergedSources[chainID][sourceType]
			assert.True(t, ok, "Merged sources should contain %s for chain ID %d", sourceType, chainID)
			assert.Equal(t, expectedURL, actualURL, "URL for %s on chain ID %d should match", sourceType, chainID)
		}
	}

	// Check SushiSources
	for chainID, sourcesMap := range SushiSources {
		for sourceType, expectedURL := range sourcesMap {
			actualURL, ok := mergedSources[chainID][sourceType]
			assert.True(t, ok, "Merged sources should contain %s for chain ID %d", sourceType, chainID)
			assert.Equal(t, expectedURL, actualURL, "URL for %s on chain ID %d should match", sourceType, chainID)
		}
	}

	// Check CoinGeckoSources
	for chainID, sourcesMap := range CoinGeckoSources {
		for sourceType, expectedURL := range sourcesMap {
			actualURL, ok := mergedSources[chainID][sourceType]
			assert.True(t, ok, "Merged sources should contain %s for chain ID %d", sourceType, chainID)
			assert.Equal(t, expectedURL, actualURL, "URL for %s on chain ID %d should match", sourceType, chainID)
		}
	}

	// Check PancakeSources
	for chainID, sourcesMap := range PancakeSources {
		for sourceType, expectedURL := range sourcesMap {
			actualURL, ok := mergedSources[chainID][sourceType]
			assert.True(t, ok, "Merged sources should contain %s for chain ID %d", sourceType, chainID)
			assert.Equal(t, expectedURL, actualURL, "URL for %s on chain ID %d should match", sourceType, chainID)
		}
	}

	// Spot check some specific entries (keeping these as additional verification)
	url, ok := mergedSources[1][SourceTypeERC20]
	assert.True(t, ok, "Merged sources should contain mainnet ERC20")
	assert.NotEmpty(t, url, "Mainnet ERC20 URL should not be empty")

	url, ok = mergedSources[1][SourceTypeUniswap]
	assert.True(t, ok, "Merged sources should contain mainnet Uniswap")
	assert.NotEmpty(t, url, "Mainnet Uniswap URL should not be empty")

	url, ok = mergedSources[56][SourceTypePancake]
	assert.True(t, ok, "Merged sources should contain BSC Pancake")
	assert.NotEmpty(t, url, "BSC Pancake URL should not be empty")
}
