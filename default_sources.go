package tokendirectory

type SourceType string

func (s SourceType) String() string {
	return string(s)
}

// List of sources for token lists.
const (
	SourceTypeERC20   SourceType = "erc20"
	SourceTypeERC721  SourceType = "erc721"
	SourceTypeERC1155 SourceType = "erc1155"
)

// _DefaultSources tokenDirectorySources, order of precedence is from top to bottom, meaning
// token info in lists higher up take precedence.
var _DefaultSources = map[uint64]map[SourceType]string{
	// mainnet
	1: {
		SourceTypeERC20:   "https://metadata.sequence.app/token-directory/mainnet/erc20.json",
		SourceTypeERC721:  "https://metadata.sequence.app/token-directory/mainnet/erc721.json",
		SourceTypeERC1155: "https://metadata.sequence.app/token-directory/mainnet/erc1155.json",
	},
	// polygon
	137: {
		SourceTypeERC20:   "https://metadata.sequence.app/token-directory/polygon/erc20.json",
		SourceTypeERC721:  "https://metadata.sequence.app/token-directory/polygon/erc721.json",
		SourceTypeERC1155: "https://metadata.sequence.app/token-directory/polygon/erc1155.json",
	},
	// polygon zkevm
	1101: {
		SourceTypeERC20:   "https://metadata.sequence.app/token-directory/polygon-zkevm/erc20.json",
		SourceTypeERC721:  "https://metadata.sequence.app/token-directory/polygon-zkevm/erc721.json",
		SourceTypeERC1155: "https://metadata.sequence.app/token-directory/polygon-zkevm/erc1155.json",
	},
	// mumbai
	80001: {
		SourceTypeERC20:   "https://metadata.sequence.app/token-directory/mumbai/erc20.json",
		SourceTypeERC721:  "https://metadata.sequence.app/token-directory/mumbai/erc721.json",
		SourceTypeERC1155: "https://metadata.sequence.app/token-directory/mumbai/erc1155.json",
	},
	// BSC
	56: {
		SourceTypeERC20:   "https://metadata.sequence.app/token-directory/bnb/erc20.json",
		SourceTypeERC721:  "https://metadata.sequence.app/token-directory/bnb/erc721.json",
		SourceTypeERC1155: "https://metadata.sequence.app/token-directory/bnb/erc1155.json",
	},
	// BSC-testnet
	97: {
		SourceTypeERC20:   "https://metadata.sequence.app/token-directory/bnb-testnet/erc20.json",
		SourceTypeERC721:  "https://metadata.sequence.app/token-directory/bnb-testnet/erc721.json",
		SourceTypeERC1155: "https://metadata.sequence.app/token-directory/bnb-testnet/erc1155.json",
	},
	// arbitrum
	42161: {
		SourceTypeERC20:   "https://metadata.sequence.app/token-directory/arbitrum/erc20.json",
		SourceTypeERC721:  "https://metadata.sequence.app/token-directory/arbitrum/erc721.json",
		SourceTypeERC1155: "https://metadata.sequence.app/token-directory/arbitrum/erc1155.json",
	},
	// arbitrum-nova
	42170: {
		SourceTypeERC20:   "https://metadata.sequence.app/token-directory/arbitrum-nova/erc20.json",
		SourceTypeERC721:  "https://metadata.sequence.app/token-directory/arbitrum-nova/erc721.json",
		SourceTypeERC1155: "https://metadata.sequence.app/token-directory/arbitrum-nova/erc1155.json",
	},
	// avalanche
	43114: {
		SourceTypeERC20:   "https://metadata.sequence.app/token-directory/avalanche/erc20.json",
		SourceTypeERC721:  "https://metadata.sequence.app/token-directory/avalanche/erc721.json",
		SourceTypeERC1155: "https://metadata.sequence.app/token-directory/avalanche/erc1155.json",
	},
	// optimism
	10: {
		SourceTypeERC20:   "https://metadata.sequence.app/token-directory/optimism/erc20.json",
		SourceTypeERC721:  "https://metadata.sequence.app/token-directory/optimism/erc721.json",
		SourceTypeERC1155: "https://metadata.sequence.app/token-directory/optimism/erc1155.json",
	},
	// gnosis
	100: {
		SourceTypeERC20:   "https://metadata.sequence.app/token-directory/gnosis/erc20.json",
		SourceTypeERC721:  "https://metadata.sequence.app/token-directory/gnosis/erc721.json",
		SourceTypeERC1155: "https://metadata.sequence.app/token-directory/gnosis/erc1155.json",
	},
	// base
	8453: {
		SourceTypeERC20:   "https://metadata.sequence.app/token-directory/base/erc20.json",
		SourceTypeERC721:  "https://metadata.sequence.app/token-directory/base/erc721.json",
		SourceTypeERC1155: "https://metadata.sequence.app/token-directory/base/erc1155.json",
	},
	// base-goerli
	84531: {
		SourceTypeERC20:   "https://metadata.sequence.app/token-directory/base-goerli/erc20.json",
		SourceTypeERC721:  "https://metadata.sequence.app/token-directory/base-goerli/erc721.json",
		SourceTypeERC1155: "https://metadata.sequence.app/token-directory/base-goerli/erc1155.json",
	},
	// homeverse
	19011: {
		SourceTypeERC20:   "https://metadata.sequence.app/token-directory/homeverse/erc20.json",
		SourceTypeERC721:  "https://metadata.sequence.app/token-directory/homeverse/erc721.json",
		SourceTypeERC1155: "https://metadata.sequence.app/token-directory/homeverse/erc1155.json",
	},
	// homeverse-testnet
	40875: {
		SourceTypeERC20:   "https://metadata.sequence.app/token-directory/homeverse-testnet/erc20.json",
		SourceTypeERC721:  "https://metadata.sequence.app/token-directory/homeverse-testnet/erc721.json",
		SourceTypeERC1155: "https://metadata.sequence.app/token-directory/homeverse-testnet/erc1155.json",
	},
	// sepolia
	11155111: {
		SourceTypeERC20:   "https://metadata.sequence.app/token-directory/sepolia/erc20.json",
		SourceTypeERC721:  "https://metadata.sequence.app/token-directory/sepolia/erc721.json",
		SourceTypeERC1155: "https://metadata.sequence.app/token-directory/sepolia/erc1155.json",
	},
	// base-sepolia
	84532: {
		SourceTypeERC20:   "https://metadata.sequence.app/token-directory/base-sepolia/erc20.json",
		SourceTypeERC721:  "https://metadata.sequence.app/token-directory/base-sepolia/erc721.json",
		SourceTypeERC1155: "https://metadata.sequence.app/token-directory/base-sepolia/erc1155.json",
	},
	// arbitrum-sepolia
	421614: {
		SourceTypeERC20:   "https://metadata.sequence.app/token-directory/arbitrum-sepolia/erc20.json",
		SourceTypeERC721:  "https://metadata.sequence.app/token-directory/arbitrum-sepolia/erc721.json",
		SourceTypeERC1155: "https://metadata.sequence.app/token-directory/arbitrum-sepolia/erc1155.json",
	},
	// optimism-sepolia
	11155420: {
		SourceTypeERC20:   "https://metadata.sequence.app/token-directory/optimism-sepolia/erc20.json",
		SourceTypeERC721:  "https://metadata.sequence.app/token-directory/optimism-sepolia/erc721.json",
		SourceTypeERC1155: "https://metadata.sequence.app/token-directory/optimism-sepolia/erc1155.json",
	},
	// astar-zkevm
	3776: {
		SourceTypeERC20:   "https://metadata.sequence.app/token-directory/astar-zkevm/erc20.json",
		SourceTypeERC721:  "https://metadata.sequence.app/token-directory/astar-zkevm/erc721.json",
		SourceTypeERC1155: "https://metadata.sequence.app/token-directory/astar-zkevm/erc1155.json",
	},
	// astar-zkyoto
	6038361: {
		SourceTypeERC20:   "https://metadata.sequence.app/token-directory/astar-zkyoto/erc20.json",
		SourceTypeERC721:  "https://metadata.sequence.app/token-directory/astar-zkyoto/erc721.json",
		SourceTypeERC1155: "https://metadata.sequence.app/token-directory/astar-zkyoto/erc1155.json",
	},
}
