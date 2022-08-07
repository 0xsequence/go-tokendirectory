package tokendirectory

// Sources from github.com/0xsequence/token-directory and other token-list sources.

// tokenDirectorySources, order of precedence is from top to bottom, meaning
// token info in lists higher up take precedence.
var DefaultTokenDirectorySources = map[uint64][]string{
	1: {
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/mainnet/erc20.json",
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/mainnet/erc721.json",
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/mainnet/erc1155.json",
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/mainnet/misc.json",
		"https://unpkg.com/@uniswap/default-token-list@2.0.0/build/uniswap-default.tokenlist.json",
		"https://unpkg.com/@sushiswap/default-token-list@16.18.0/build/sushiswap-default.tokenlist.json",
		"https://tokens.coingecko.com/uniswap/all.json",
	},
	137: {
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/polygon/erc20.json",
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/polygon/erc721.json",
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/polygon/erc1155.json",
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/polygon/misc.json",
		"https://unpkg.com/@sushiswap/default-token-list@16.18.0/build/sushiswap-default.tokenlist.json",
	},
	4: {
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/rinkeby/erc20.json",
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/rinkeby/erc1155.json",
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/rinkeby/misc.json",
	},
	80001: {
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/mumbai/erc20.json",
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/mumbai/erc721.json",
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/mumbai/erc1155.json",
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/mumbai/misc.json",
	},
	56: {
		"https://raw.githubusercontent.com/0xsequence/token-directory/master/index/bnb/erc20.json",
		"https://raw.githubusercontent.com/pancakeswap/pancake-toolkit/master/packages/token-lists/lists/pancakeswap-default.json",
	},
}
