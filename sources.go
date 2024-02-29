package tokendirectory

import (
	"slices"
)

// Source from github.com/0xsequence/token-directory and other token-list sources.
type Source interface {
	GetID() string
	GetChainIDs() []uint64
	GetURLs(chainID uint64) []string
}

type StaticSource map[uint64][]string

func (s StaticSource) GetID() string {
	return "static"

}

func (s StaticSource) GetChainIDs() []uint64 {
	chainIDs := make([]uint64, 0, len(s))
	for chainID := range s {
		chainIDs = append(chainIDs, chainID)
	}
	slices.Sort(chainIDs)
	return chainIDs
}

func (s StaticSource) GetURLs(chainID uint64) []string {
	return s[chainID]
}

// DefaultSources tokenDirectorySources, order of precedence is from top to bottom, meaning
// token info in lists higher up take precedence.
var DefaultSources Source = StaticSource{
	// mainnet
	1: {
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/mainnet/erc20.json",
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/mainnet/erc721.json",
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/mainnet/erc1155.json",
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/mainnet/misc.json",
		"https://unpkg.com/@uniswap/default-token-list@4.1.0/build/uniswap-default.tokenlist.json",
		"https://unpkg.com/@sushiswap/default-token-list@34.0.0/build/sushiswap-default.tokenlist.json",
		"https://tokens.coingecko.com/uniswap/all.json",
	},
	// polygon
	137: {
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/polygon/erc20.json",
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/polygon/erc721.json",
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/polygon/erc1155.json",
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/polygon/misc.json",
		"https://unpkg.com/@sushiswap/default-token-list@34.0.0/build/sushiswap-default.tokenlist.json",
	},
	// polygon zkevm
	1101: {
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/polygon-zkevm/erc20.json",
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/polygon-zkevm/erc721.json",
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/polygon-zkevm/erc1155.json",
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/polygon-zkevm/misc.json",
	},
	// goerli
	5: {
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/goerli/erc20.json",
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/goerli/erc721.json",
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/goerli/erc1155.json",
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/goerli/misc.json",
	},
	// mumbai
	80001: {
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/mumbai/erc20.json",
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/mumbai/erc721.json",
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/mumbai/erc1155.json",
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/mumbai/misc.json",
	},
	// BSC
	56: {
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/bnb/erc20.json",
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/bnb/erc721.json",
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/bnb/erc1155.json",
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/bnb/misc.json",
		"https://raw.githubusercontent.com/pancakeswap/pancake-toolkit/master/packages/token-lists/lists/pancakeswap-default.json",
	},
	// BSC-testnet
	97: {
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/bnb-testnet/erc20.json",
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/bnb-testnet/erc721.json",
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/bnb-testnet/erc1155.json",
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/bnb-testnet/misc.json",
	},
	// arbitrum
	42161: {
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/arbitrum/erc20.json",
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/arbitrum/erc721.json",
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/arbitrum/erc1155.json",
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/arbitrum/misc.json",
	},
	// arbitrum-nova
	42170: {
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/arbitrum-nova/erc20.json",
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/arbitrum-nova/erc721.json",
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/arbitrum-nova/erc1155.json",
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/arbitrum-nova/misc.json",
		"https://raw.githubusercontent.com/sushiswap/list/master/lists/token-lists/default-token-list/tokens/arbitrum-nova.json",
	},
	// avalanche
	43114: {
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/avalanche/erc20.json",
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/avalanche/erc721.json",
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/avalanche/erc1155.json",
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/avalanche/misc.json",
		"https://raw.githubusercontent.com/sushiswap/list/master/lists/token-lists/default-token-list/tokens/avalanche.json",
	},
	// optimism
	10: {
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/optimism/erc20.json",
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/optimism/erc721.json",
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/optimism/erc1155.json",
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/optimism/misc.json",
	},
	// gnosis
	100: {
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/gnosis/erc20.json",
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/gnosis/erc721.json",
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/gnosis/erc1155.json",
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/gnosis/misc.json",
		"https://raw.githubusercontent.com/sushiswap/list/master/lists/token-lists/default-token-list/tokens/xdai.json",
	},
	// base
	8453: {
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/base/erc20.json",
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/base/erc721.json",
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/base/erc1155.json",
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/base/misc.json",
	},
	// base-goerli
	84531: {
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/base-goerli/erc20.json",
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/base-goerli/erc721.json",
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/base-goerli/erc1155.json",
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/base-goerli/misc.json",
	},
	// homeverse
	19011: {
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/homeverse/erc20.json",
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/homeverse/erc721.json",
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/homeverse/erc1155.json",
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/homeverse/misc.json",
	},
	// homeverse-testnet
	40875: {
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/homeverse-testnet/erc20.json",
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/homeverse-testnet/erc721.json",
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/homeverse-testnet/erc1155.json",
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/homeverse-testnet/misc.json",
	},
	// sepolia
	11155111: {
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/sepolia/erc20.json",
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/sepolia/erc721.json",
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/sepolia/erc1155.json",
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/sepolia/misc.json",
	},
	// base-sepolia
	84532: {
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/base-sepolia/erc20.json",
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/base-sepolia/erc721.json",
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/base-sepolia/erc1155.json",
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/base-sepolia/misc.json",
	},
	// arbitrum-sepolia
	421614: {
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/arbitrum-sepolia/erc20.json",
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/arbitrum-sepolia/erc721.json",
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/arbitrum-sepolia/erc1155.json",
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/arbitrum-sepolia/misc.json",
	},
	// optimism-sepolia
	11155420: {
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/optimism-sepolia/erc20.json",
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/optimism-sepolia/erc721.json",
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/optimism-sepolia/erc1155.json",
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/optimism-sepolia/misc.json",
	},
}
