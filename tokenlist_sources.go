package tokendirectory

import "maps"

type SourceType string

func (s SourceType) String() string {
	return string(s)
}

// List of sources for token lists.
const (
	SourceTypeERC20     SourceType = "erc20"
	SourceTypeERC721    SourceType = "erc721"
	SourceTypeERC1155   SourceType = "erc1155"
	SourceTypeMisc      SourceType = "misc"
	SourceTypeUniswap   SourceType = "uniswap"
	SourceTypeSushi     SourceType = "sushiswap"
	SourceTypePancake   SourceType = "pancakeswap"
	SourceTypeCoinGecko SourceType = "coingecko"
)

// TODO: for sequence github sources, we can make this dynamic on sync, and remove the need for this map
// as the routes are deterministic .. we just need to expand our data structures on the actual token-directory
// such as, providing an index of all chain names and ids, and then a list of all files, etc.
// pretty much, this map exactly can be included in the token-directory and then we'll have a "index.json"
//
// in fact, we can even do that for token lists like uniswap, sushi, etc. too..

// SequenceGithubSources tokenDirectorySources, order of precedence is from top to bottom, meaning
// token info in lists higher up take precedence.
var SequenceGithubSources = map[uint64]map[SourceType]string{
	// mainnet
	1: {
		SourceTypeERC20:   "https://raw.githubusercontent.com/0xsequence/token-directory/master/index/mainnet/erc20.json",
		SourceTypeERC721:  "https://raw.githubusercontent.com/0xsequence/token-directory/master/index/mainnet/erc721.json",
		SourceTypeERC1155: "https://raw.githubusercontent.com/0xsequence/token-directory/master/index/mainnet/erc1155.json",
		SourceTypeMisc:    "https://raw.githubusercontent.com/0xsequence/token-directory/master/index/mainnet/misc.json",
	},
	// polygon
	137: {
		SourceTypeERC20:   "https://raw.githubusercontent.com/0xsequence/token-directory/master/index/polygon/erc20.json",
		SourceTypeERC721:  "https://raw.githubusercontent.com/0xsequence/token-directory/master/index/polygon/erc721.json",
		SourceTypeERC1155: "https://raw.githubusercontent.com/0xsequence/token-directory/master/index/polygon/erc1155.json",
		SourceTypeMisc:    "https://raw.githubusercontent.com/0xsequence/token-directory/master/index/polygon/misc.json",
	},
	// polygon zkevm
	1101: {
		SourceTypeERC20:   "https://raw.githubusercontent.com/0xsequence/token-directory/master/index/polygon-zkevm/erc20.json",
		SourceTypeERC721:  "https://raw.githubusercontent.com/0xsequence/token-directory/master/index/polygon-zkevm/erc721.json",
		SourceTypeERC1155: "https://raw.githubusercontent.com/0xsequence/token-directory/master/index/polygon-zkevm/erc1155.json",
		SourceTypeMisc:    "https://raw.githubusercontent.com/0xsequence/token-directory/master/index/polygon-zkevm/misc.json",
	},
	// mumbai (note: mumbai is deprecated)
	80001: {
		SourceTypeERC20:   "https://raw.githubusercontent.com/0xsequence/token-directory/master/index/mumbai/erc20.json",
		SourceTypeERC721:  "https://raw.githubusercontent.com/0xsequence/token-directory/master/index/mumbai/erc721.json",
		SourceTypeERC1155: "https://raw.githubusercontent.com/0xsequence/token-directory/master/index/mumbai/erc1155.json",
		SourceTypeMisc:    "https://raw.githubusercontent.com/0xsequence/token-directory/master/index/mumbai/misc.json",
	},
	80002: {
		SourceTypeERC20: "https://raw.githubusercontent.com/0xsequence/token-directory/master/index/amoy/erc20.json",
	},
	// BSC
	56: {
		SourceTypeERC20:   "https://raw.githubusercontent.com/0xsequence/token-directory/master/index/bnb/erc20.json",
		SourceTypeERC721:  "https://raw.githubusercontent.com/0xsequence/token-directory/master/index/bnb/erc721.json",
		SourceTypeERC1155: "https://raw.githubusercontent.com/0xsequence/token-directory/master/index/bnb/erc1155.json",
		SourceTypeMisc:    "https://raw.githubusercontent.com/0xsequence/token-directory/master/index/bnb/misc.json",
	},
	// BSC-testnet
	97: {
		SourceTypeERC20:   "https://raw.githubusercontent.com/0xsequence/token-directory/master/index/bnb-testnet/erc20.json",
		SourceTypeERC721:  "https://raw.githubusercontent.com/0xsequence/token-directory/master/index/bnb-testnet/erc721.json",
		SourceTypeERC1155: "https://raw.githubusercontent.com/0xsequence/token-directory/master/index/bnb-testnet/erc1155.json",
		SourceTypeMisc:    "https://raw.githubusercontent.com/0xsequence/token-directory/master/index/bnb-testnet/misc.json",
	},
	// arbitrum
	42161: {
		SourceTypeERC20:   "https://raw.githubusercontent.com/0xsequence/token-directory/master/index/arbitrum/erc20.json",
		SourceTypeERC721:  "https://raw.githubusercontent.com/0xsequence/token-directory/master/index/arbitrum/erc721.json",
		SourceTypeERC1155: "https://raw.githubusercontent.com/0xsequence/token-directory/master/index/arbitrum/erc1155.json",
		SourceTypeMisc:    "https://raw.githubusercontent.com/0xsequence/token-directory/master/index/arbitrum/misc.json",
	},
	// arbitrum-nova
	42170: {
		SourceTypeERC20:   "https://raw.githubusercontent.com/0xsequence/token-directory/master/index/arbitrum-nova/erc20.json",
		SourceTypeERC721:  "https://raw.githubusercontent.com/0xsequence/token-directory/master/index/arbitrum-nova/erc721.json",
		SourceTypeERC1155: "https://raw.githubusercontent.com/0xsequence/token-directory/master/index/arbitrum-nova/erc1155.json",
		SourceTypeMisc:    "https://raw.githubusercontent.com/0xsequence/token-directory/master/index/arbitrum-nova/misc.json",
	},
	// avalanche
	43114: {
		SourceTypeERC20:   "https://raw.githubusercontent.com/0xsequence/token-directory/master/index/avalanche/erc20.json",
		SourceTypeERC721:  "https://raw.githubusercontent.com/0xsequence/token-directory/master/index/avalanche/erc721.json",
		SourceTypeERC1155: "https://raw.githubusercontent.com/0xsequence/token-directory/master/index/avalanche/erc1155.json",
		SourceTypeMisc:    "https://raw.githubusercontent.com/0xsequence/token-directory/master/index/avalanche/misc.json",
	},
	// optimism
	10: {
		SourceTypeERC20:   "https://raw.githubusercontent.com/0xsequence/token-directory/master/index/optimism/erc20.json",
		SourceTypeERC721:  "https://raw.githubusercontent.com/0xsequence/token-directory/master/index/optimism/erc721.json",
		SourceTypeERC1155: "https://raw.githubusercontent.com/0xsequence/token-directory/master/index/optimism/erc1155.json",
		SourceTypeMisc:    "https://raw.githubusercontent.com/0xsequence/token-directory/master/index/optimism/misc.json",
	},
	// gnosis
	100: {
		SourceTypeERC20:   "https://raw.githubusercontent.com/0xsequence/token-directory/master/index/gnosis/erc20.json",
		SourceTypeERC721:  "https://raw.githubusercontent.com/0xsequence/token-directory/master/index/gnosis/erc721.json",
		SourceTypeERC1155: "https://raw.githubusercontent.com/0xsequence/token-directory/master/index/gnosis/erc1155.json",
		SourceTypeMisc:    "https://raw.githubusercontent.com/0xsequence/token-directory/master/index/gnosis/misc.json",
	},
	// base
	8453: {
		SourceTypeERC20:   "https://raw.githubusercontent.com/0xsequence/token-directory/master/index/base/erc20.json",
		SourceTypeERC721:  "https://raw.githubusercontent.com/0xsequence/token-directory/master/index/base/erc721.json",
		SourceTypeERC1155: "https://raw.githubusercontent.com/0xsequence/token-directory/master/index/base/erc1155.json",
		SourceTypeMisc:    "https://raw.githubusercontent.com/0xsequence/token-directory/master/index/base/misc.json",
	},
	// base-goerli
	84531: {
		SourceTypeERC20:   "https://raw.githubusercontent.com/0xsequence/token-directory/master/index/base-goerli/erc20.json",
		SourceTypeERC721:  "https://raw.githubusercontent.com/0xsequence/token-directory/master/index/base-goerli/erc721.json",
		SourceTypeERC1155: "https://raw.githubusercontent.com/0xsequence/token-directory/master/index/base-goerli/erc1155.json",
		SourceTypeMisc:    "https://raw.githubusercontent.com/0xsequence/token-directory/master/index/base-goerli/misc.json",
	},
	// homeverse
	19011: {
		SourceTypeERC20:   "https://raw.githubusercontent.com/0xsequence/token-directory/master/index/homeverse/erc20.json",
		SourceTypeERC721:  "https://raw.githubusercontent.com/0xsequence/token-directory/master/index/homeverse/erc721.json",
		SourceTypeERC1155: "https://raw.githubusercontent.com/0xsequence/token-directory/master/index/homeverse/erc1155.json",
		SourceTypeMisc:    "https://raw.githubusercontent.com/0xsequence/token-directory/master/index/homeverse/misc.json",
	},
	// homeverse-testnet
	40875: {
		SourceTypeERC20:   "https://raw.githubusercontent.com/0xsequence/token-directory/master/index/homeverse-testnet/erc20.json",
		SourceTypeERC721:  "https://raw.githubusercontent.com/0xsequence/token-directory/master/index/homeverse-testnet/erc721.json",
		SourceTypeERC1155: "https://raw.githubusercontent.com/0xsequence/token-directory/master/index/homeverse-testnet/erc1155.json",
		SourceTypeMisc:    "https://raw.githubusercontent.com/0xsequence/token-directory/master/index/homeverse-testnet/misc.json",
	},
	// sepolia
	11155111: {
		SourceTypeERC20:   "https://raw.githubusercontent.com/0xsequence/token-directory/master/index/sepolia/erc20.json",
		SourceTypeERC721:  "https://raw.githubusercontent.com/0xsequence/token-directory/master/index/sepolia/erc721.json",
		SourceTypeERC1155: "https://raw.githubusercontent.com/0xsequence/token-directory/master/index/sepolia/erc1155.json",
		SourceTypeMisc:    "https://raw.githubusercontent.com/0xsequence/token-directory/master/index/sepolia/misc.json",
	},
	// base-sepolia
	84532: {
		SourceTypeERC20:   "https://raw.githubusercontent.com/0xsequence/token-directory/master/index/base-sepolia/erc20.json",
		SourceTypeERC721:  "https://raw.githubusercontent.com/0xsequence/token-directory/master/index/base-sepolia/erc721.json",
		SourceTypeERC1155: "https://raw.githubusercontent.com/0xsequence/token-directory/master/index/base-sepolia/erc1155.json",
		SourceTypeMisc:    "https://raw.githubusercontent.com/0xsequence/token-directory/master/index/base-sepolia/misc.json",
	},
	// arbitrum-sepolia
	421614: {
		SourceTypeERC20:   "https://raw.githubusercontent.com/0xsequence/token-directory/master/index/arbitrum-sepolia/erc20.json",
		SourceTypeERC721:  "https://raw.githubusercontent.com/0xsequence/token-directory/master/index/arbitrum-sepolia/erc721.json",
		SourceTypeERC1155: "https://raw.githubusercontent.com/0xsequence/token-directory/master/index/arbitrum-sepolia/erc1155.json",
		SourceTypeMisc:    "https://raw.githubusercontent.com/0xsequence/token-directory/master/index/arbitrum-sepolia/misc.json",
	},
	// optimism-sepolia
	11155420: {
		SourceTypeERC20:   "https://raw.githubusercontent.com/0xsequence/token-directory/master/index/optimism-sepolia/erc20.json",
		SourceTypeERC721:  "https://raw.githubusercontent.com/0xsequence/token-directory/master/index/optimism-sepolia/erc721.json",
		SourceTypeERC1155: "https://raw.githubusercontent.com/0xsequence/token-directory/master/index/optimism-sepolia/erc1155.json",
		SourceTypeMisc:    "https://raw.githubusercontent.com/0xsequence/token-directory/master/index/optimism-sepolia/misc.json",
	},
	// astar-zkevm
	3776: {
		SourceTypeERC20:   "https://raw.githubusercontent.com/0xsequence/token-directory/master/index/astar-zkevm/erc20.json",
		SourceTypeERC721:  "https://raw.githubusercontent.com/0xsequence/token-directory/master/index/astar-zkevm/erc721.json",
		SourceTypeERC1155: "https://raw.githubusercontent.com/0xsequence/token-directory/master/index/astar-zkevm/erc1155.json",
		SourceTypeMisc:    "https://raw.githubusercontent.com/0xsequence/token-directory/master/index/astar-zkevm/misc.json",
	},
	// astar-zkyoto
	6038361: {
		SourceTypeERC20:   "https://raw.githubusercontent.com/0xsequence/token-directory/master/index/astar-zkyoto/erc20.json",
		SourceTypeERC721:  "https://raw.githubusercontent.com/0xsequence/token-directory/master/index/astar-zkyoto/erc721.json",
		SourceTypeERC1155: "https://raw.githubusercontent.com/0xsequence/token-directory/master/index/astar-zkyoto/erc1155.json",
		SourceTypeMisc:    "https://raw.githubusercontent.com/0xsequence/token-directory/master/index/astar-zkyoto/misc.json",
	},
}

var UniswapSources = map[uint64]map[SourceType]string{
	// mainnet
	1: {
		SourceTypeUniswap: "https://unpkg.com/@uniswap/default-token-list@4.1.0/build/uniswap-default.tokenlist.json",
	},
}

var SushiSources = map[uint64]map[SourceType]string{
	// mainnet
	1: {
		SourceTypeSushi: "https://unpkg.com/@sushiswap/default-token-list@34.0.0/build/sushiswap-default.tokenlist.json",
	},
	// polygon
	137: {
		SourceTypeSushi: "https://unpkg.com/@sushiswap/default-token-list@34.0.0/build/sushiswap-default.tokenlist.json",
	},
	// arbitrum-nova
	42170: {
		SourceTypeSushi: "https://raw.githubusercontent.com/sushiswap/list/master/lists/token-lists/default-token-list/tokens/arbitrum-nova.json",
	},
	// avalanche
	43114: {
		SourceTypeSushi: "https://raw.githubusercontent.com/sushiswap/list/master/lists/token-lists/default-token-list/tokens/avalanche.json",
	},
	// gnosis
	100: {
		SourceTypeSushi: "https://raw.githubusercontent.com/sushiswap/list/master/lists/token-lists/default-token-list/tokens/xdai.json",
	},
}

var CoinGeckoSources = map[uint64]map[SourceType]string{
	// mainnet
	1: {
		SourceTypeCoinGecko: "https://tokens.coingecko.com/uniswap/all.json",
	},
}

var PancakeSources = map[uint64]map[SourceType]string{
	// BSC
	56: {
		SourceTypePancake: "https://raw.githubusercontent.com/pancakeswap/pancake-toolkit/master/packages/token-lists/lists/pancakeswap-default.json",
	},
}

func MergeSources(sources ...map[uint64]map[SourceType]string) map[uint64]map[SourceType]string {
	mergedSources := make(map[uint64]map[SourceType]string)
	for _, source := range sources {
		for chainID, chainSources := range source {
			s, ok := mergedSources[chainID]
			if !ok {
				mergedSources[chainID] = make(map[SourceType]string)
				maps.Copy(mergedSources[chainID], chainSources)
			} else {
				for sourceType, sourceURL := range chainSources {
					s[sourceType] = sourceURL
				}
			}
		}
	}
	return mergedSources
}
