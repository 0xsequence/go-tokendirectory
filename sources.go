package tokendirectory

// Sources from github.com/0xsequence/token-directory and other token-list sources.

type Sources map[uint64][]string

// DefaultSources tokenDirectorySources, order of precedence is from top to bottom, meaning
// token info in lists higher up take precedence. Listed in alphabetical order by chain name.
var DefaultSources Sources = map[uint64][]string{
	// arbitrum
	42161: {
		"https://api.sequence.build/token-directory/arbitrum",
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/arbitrum/erc20.json",
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/arbitrum/erc721.json",
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/arbitrum/erc1155.json",
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/arbitrum/misc.json",
	},

	// arbitrum-goerli
	421613: {
		"https://api.sequence.build/token-directory/arbitrum-goerli",
	},

	// arbitrum-nova
	42170: {
		"https://api.sequence.build/token-directory/arbitrum-nova",
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/arbitrum-nova/erc20.json",
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/arbitrum-nova/erc721.json",
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/arbitrum-nova/erc1155.json",
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/arbitrum-nova/misc.json",
		"https://raw.githubusercontent.com/sushiswap/list/master/lists/token-lists/default-token-list/tokens/arbitrum-nova.json",
	},

	// arbitrum-sepolia
	421614: {
		"https://api.sequence.build/token-directory/arbitrum-sepolia",
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/arbitrum-sepolia/erc20.json",
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/arbitrum-sepolia/erc721.json",
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/arbitrum-sepolia/erc1155.json",
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/arbitrum-sepolia/misc.json",
	},

	// avalanche
	43114: {
		"https://api.sequence.build/token-directory/avalanche",
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/avalanche/erc20.json",
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/avalanche/erc721.json",
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/avalanche/erc1155.json",
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/avalanche/misc.json",
		"https://raw.githubusercontent.com/sushiswap/list/master/lists/token-lists/default-token-list/tokens/avalanche.json",
	},

	// avalanche-testnet
	43113: {
		"https://api.sequence.build/token-directory/43113",
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/avalanche-testnet/erc20.json",
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/avalanche-testnet/erc721.json",
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/avalanche-testnet/erc1155.json",
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/avalanche-testnet/misc.json",
	},

	// base
	8453: {
		"https://api.sequence.build/token-directory/base",
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/base/erc20.json",
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/base/erc721.json",
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/base/erc1155.json",
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/base/misc.json",
	},

	// base-goerli
	84531: {
		"https://api.sequence.build/token-directory/84531",
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/base-goerli/erc20.json",
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/base-goerli/erc721.json",
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/base-goerli/erc1155.json",
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/base-goerli/misc.json",
	},

	// base-sepolia
	84532: {
		"https://api.sequence.build/token-directory/base-sepolia",
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/base-sepolia/erc20.json",
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/base-sepolia/erc721.json",
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/base-sepolia/erc1155.json",
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/base-sepolia/misc.json",
	},

	// bsc
	56: {
		"https://api.sequence.build/token-directory/bsc",
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/bnb/erc20.json",
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/bnb/erc721.json",
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/bnb/erc1155.json",
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/bnb/misc.json",
		"https://raw.githubusercontent.com/pancakeswap/pancake-toolkit/master/packages/token-lists/lists/pancakeswap-default.json",
	},

	// bsc-testnet
	97: {
		"https://api.sequence.build/token-directory/bsc-testnet",
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/bnb-testnet/erc20.json",
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/bnb-testnet/erc721.json",
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/bnb-testnet/erc1155.json",
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/bnb-testnet/misc.json",
	},

	// gnosis
	100: {
		"https://api.sequence.build/token-directory/gnosis",
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/gnosis/erc20.json",
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/gnosis/erc721.json",
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/gnosis/erc1155.json",
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/gnosis/misc.json",
		"https://raw.githubusercontent.com/sushiswap/list/master/lists/token-lists/default-token-list/tokens/xdai.json",
	},

	// goerli
	5: {
		"https://api.sequence.build/token-directory/5",
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/goerli/erc20.json",
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/goerli/erc721.json",
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/goerli/erc1155.json",
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/goerli/misc.json",
	},

	// homeverse
	19011: {
		"https://api.sequence.build/token-directory/homeverse",
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/homeverse/erc20.json",
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/homeverse/erc721.json",
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/homeverse/erc1155.json",
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/homeverse/misc.json",
	},

	// homeverse-testnet
	40875: {
		"https://api.sequence.build/token-directory/homeverse-testnet",
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/homeverse-testnet/erc20.json",
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/homeverse-testnet/erc721.json",
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/homeverse-testnet/erc1155.json",
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/homeverse-testnet/misc.json",
	},

	// mainnet
	1: {
		"https://api.sequence.build/token-directory/mainnet",
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/mainnet/erc20.json",
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/mainnet/erc721.json",
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/mainnet/erc1155.json",
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/mainnet/misc.json",
		"https://unpkg.com/@uniswap/default-token-list@4.1.0/build/uniswap-default.tokenlist.json",
		"https://unpkg.com/@sushiswap/default-token-list@34.0.0/build/sushiswap-default.tokenlist.json",
		"https://tokens.coingecko.com/uniswap/all.json",
	},

	// mumbai
	80001: {
		"https://api.sequence.build/token-directory/mumbai",
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/mumbai/erc20.json",
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/mumbai/erc721.json",
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/mumbai/erc1155.json",
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/mumbai/misc.json",
	},

	// optimism
	10: {
		"https://api.sequence.build/token-directory/optimism",
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/optimism/erc20.json",
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/optimism/erc721.json",
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/optimism/erc1155.json",
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/optimism/misc.json",
	},

	// optimism-sepolia
	11155420: {
		"https://api.sequence.build/token-directory/optimism-sepolia",
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/optimism-sepolia/erc20.json",
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/optimism-sepolia/erc721.json",
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/optimism-sepolia/erc1155.json",
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/optimism-sepolia/misc.json",
	},

	// polygon
	137: {
		"https://api.sequence.build/token-directory/polygon",
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/polygon/erc20.json",
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/polygon/erc721.json",
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/polygon/erc1155.json",
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/polygon/misc.json",
		"https://unpkg.com/@sushiswap/default-token-list@34.0.0/build/sushiswap-default.tokenlist.json",
	},

	// polygon-zkevm
	1101: {
		"https://api.sequence.build/token-directory/polygon-zkevm",
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/polygon-zkevm/erc20.json",
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/polygon-zkevm/erc721.json",
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/polygon-zkevm/erc1155.json",
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/polygon-zkevm/misc.json",
	},

	// sepolia
	11155111: {
		"https://api.sequence.build/token-directory/sepolia",
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/sepolia/erc20.json",
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/sepolia/erc721.json",
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/sepolia/erc1155.json",
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/sepolia/misc.json",
	},
}
