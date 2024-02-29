package tokendirectory

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"slices"
)

// Provider is a source of tokens, organized by chainID and sourceName.
type Provider interface {
	GetID() string
	GetChainIDs() []uint64
	GetSources(chainID uint64) []string
	FetchTokenList(ctx context.Context, chainID uint64, sourceName string) (*TokenList, error)
}

type defaultProvider struct {
	client *http.Client
}

func (defaultProvider) GetID() string {
	return "token-directory"

}

func (defaultProvider) GetChainIDs() []uint64 {
	chainIDs := make([]uint64, 0, len(defaultSources))
	for chainID := range defaultSources {
		chainIDs = append(chainIDs, chainID)
	}
	slices.Sort(chainIDs)
	return chainIDs
}

func (defaultProvider) GetSources(chainID uint64) []string {
	sources := make([]string, 0, len(defaultSources[chainID]))
	for source := range defaultSources[chainID] {
		sources = append(sources, source)
	}
	return sources
}

func (p defaultProvider) FetchTokenList(ctx context.Context, chainID uint64, sourceName string) (*TokenList, error) {
	req, err := http.NewRequest("GET", defaultSources[chainID][sourceName], nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	res, err := p.client.Do(req.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("fetching: %w", err)
	}
	defer res.Body.Close()

	buf, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("reading body: %w", err)
	}

	var list TokenList
	if err := json.Unmarshal(buf, &list); err != nil {
		// failed to decode, likely because it doesn't follow standard token-list format,
		// and its just returning the ".tokens" part.
		list = TokenList{Name: fmt.Sprintf("%d", chainID), ChainID: chainID}

		tokens := list.Tokens
		err = json.Unmarshal(buf, &tokens)
		if err != nil {
			return nil, fmt.Errorf("decoding json: %w", err)
		}
		list.Tokens = tokens
	}

	return &list, nil
}

// List of sources for token lists.
const (
	ERC20     = "erc20"
	ERC721    = "erc721"
	ERC1155   = "erc1155"
	Misc      = "misc"
	Uniswap   = "uniswap"
	Sushi     = "sushiswap"
	Pancake   = "pancakeswap"
	CoinGecko = "coingecko"
)

// defaultSources tokenDirectorySources, order of precedence is from top to bottom, meaning
// token info in lists higher up take precedence.
var defaultSources = map[uint64]map[string]string{
	// mainnet
	1: {
		ERC20:     "https://raw.githubusercontent.com/0xsequence/token-directory/master/index/mainnet/erc20.json",
		ERC721:    "https://raw.githubusercontent.com/0xsequence/token-directory/master/index/mainnet/erc721.json",
		ERC1155:   "https://raw.githubusercontent.com/0xsequence/token-directory/master/index/mainnet/erc1155.json",
		Misc:      "https://raw.githubusercontent.com/0xsequence/token-directory/master/index/mainnet/misc.json",
		Uniswap:   "https://unpkg.com/@uniswap/default-token-list@4.1.0/build/uniswap-default.tokenlist.json",
		Sushi:     "https://unpkg.com/@sushiswap/default-token-list@34.0.0/build/sushiswap-default.tokenlist.json",
		CoinGecko: "https://tokens.coingecko.com/uniswap/all.json",
	},
	// polygon
	137: {
		ERC20:   "https://raw.githubusercontent.com/0xsequence/token-directory/master/index/polygon/erc20.json",
		ERC721:  "https://raw.githubusercontent.com/0xsequence/token-directory/master/index/polygon/erc721.json",
		ERC1155: "https://raw.githubusercontent.com/0xsequence/token-directory/master/index/polygon/erc1155.json",
		Misc:    "https://raw.githubusercontent.com/0xsequence/token-directory/master/index/polygon/misc.json",
		Sushi:   "https://unpkg.com/@sushiswap/default-token-list@34.0.0/build/sushiswap-default.tokenlist.json",
	},
	// polygon zkevm
	1101: {
		ERC20:   "https://raw.githubusercontent.com/0xsequence/token-directory/master/index/polygon-zkevm/erc20.json",
		ERC721:  "https://raw.githubusercontent.com/0xsequence/token-directory/master/index/polygon-zkevm/erc721.json",
		ERC1155: "https://raw.githubusercontent.com/0xsequence/token-directory/master/index/polygon-zkevm/erc1155.json",
		Misc:    "https://raw.githubusercontent.com/0xsequence/token-directory/master/index/polygon-zkevm/misc.json",
	},
	// goerli
	5: {
		ERC20:   "https://raw.githubusercontent.com/0xsequence/token-directory/master/index/goerli/erc20.json",
		ERC721:  "https://raw.githubusercontent.com/0xsequence/token-directory/master/index/goerli/erc721.json",
		ERC1155: "https://raw.githubusercontent.com/0xsequence/token-directory/master/index/goerli/erc1155.json",
		Misc:    "https://raw.githubusercontent.com/0xsequence/token-directory/master/index/goerli/misc.json",
	},
	// mumbai
	80001: {
		ERC20:   "https://raw.githubusercontent.com/0xsequence/token-directory/master/index/mumbai/erc20.json",
		ERC721:  "https://raw.githubusercontent.com/0xsequence/token-directory/master/index/mumbai/erc721.json",
		ERC1155: "https://raw.githubusercontent.com/0xsequence/token-directory/master/index/mumbai/erc1155.json",
		Misc:    "https://raw.githubusercontent.com/0xsequence/token-directory/master/index/mumbai/misc.json",
	},
	// BSC
	56: {
		ERC20:   "https://raw.githubusercontent.com/0xsequence/token-directory/master/index/bnb/erc20.json",
		ERC721:  "https://raw.githubusercontent.com/0xsequence/token-directory/master/index/bnb/erc721.json",
		ERC1155: "https://raw.githubusercontent.com/0xsequence/token-directory/master/index/bnb/erc1155.json",
		Misc:    "https://raw.githubusercontent.com/0xsequence/token-directory/master/index/bnb/misc.json",
		Pancake: "https://raw.githubusercontent.com/pancakeswap/pancake-toolkit/master/packages/token-lists/lists/pancakeswap-default.json",
	},
	// BSC-testnet
	97: {
		ERC20:   "https://raw.githubusercontent.com/0xsequence/token-directory/master/index/bnb-testnet/erc20.json",
		ERC721:  "https://raw.githubusercontent.com/0xsequence/token-directory/master/index/bnb-testnet/erc721.json",
		ERC1155: "https://raw.githubusercontent.com/0xsequence/token-directory/master/index/bnb-testnet/erc1155.json",
		Misc:    "https://raw.githubusercontent.com/0xsequence/token-directory/master/index/bnb-testnet/misc.json",
	},
	// arbitrum
	42161: {
		ERC20:   "https://raw.githubusercontent.com/0xsequence/token-directory/master/index/arbitrum/erc20.json",
		ERC721:  "https://raw.githubusercontent.com/0xsequence/token-directory/master/index/arbitrum/erc721.json",
		ERC1155: "https://raw.githubusercontent.com/0xsequence/token-directory/master/index/arbitrum/erc1155.json",
		Misc:    "https://raw.githubusercontent.com/0xsequence/token-directory/master/index/arbitrum/misc.json",
	},
	// arbitrum-nova
	42170: {
		ERC20:   "https://raw.githubusercontent.com/0xsequence/token-directory/master/index/arbitrum-nova/erc20.json",
		ERC721:  "https://raw.githubusercontent.com/0xsequence/token-directory/master/index/arbitrum-nova/erc721.json",
		ERC1155: "https://raw.githubusercontent.com/0xsequence/token-directory/master/index/arbitrum-nova/erc1155.json",
		Misc:    "https://raw.githubusercontent.com/0xsequence/token-directory/master/index/arbitrum-nova/misc.json",
		Sushi:   "https://raw.githubusercontent.com/sushiswap/list/master/lists/token-lists/default-token-list/tokens/arbitrum-nova.json",
	},
	// avalanche
	43114: {
		ERC20:   "https://raw.githubusercontent.com/0xsequence/token-directory/master/index/avalanche/erc20.json",
		ERC721:  "https://raw.githubusercontent.com/0xsequence/token-directory/master/index/avalanche/erc721.json",
		ERC1155: "https://raw.githubusercontent.com/0xsequence/token-directory/master/index/avalanche/erc1155.json",
		Misc:    "https://raw.githubusercontent.com/0xsequence/token-directory/master/index/avalanche/misc.json",
		Sushi:   "https://raw.githubusercontent.com/sushiswap/list/master/lists/token-lists/default-token-list/tokens/avalanche.json",
	},
	// optimism
	10: {
		ERC20:   "https://raw.githubusercontent.com/0xsequence/token-directory/master/index/optimism/erc20.json",
		ERC721:  "https://raw.githubusercontent.com/0xsequence/token-directory/master/index/optimism/erc721.json",
		ERC1155: "https://raw.githubusercontent.com/0xsequence/token-directory/master/index/optimism/erc1155.json",
		Misc:    "https://raw.githubusercontent.com/0xsequence/token-directory/master/index/optimism/misc.json",
	},
	// gnosis
	100: {
		ERC20:   "https://raw.githubusercontent.com/0xsequence/token-directory/master/index/gnosis/erc20.json",
		ERC721:  "https://raw.githubusercontent.com/0xsequence/token-directory/master/index/gnosis/erc721.json",
		ERC1155: "https://raw.githubusercontent.com/0xsequence/token-directory/master/index/gnosis/erc1155.json",
		Misc:    "https://raw.githubusercontent.com/0xsequence/token-directory/master/index/gnosis/misc.json",
		Sushi:   "https://raw.githubusercontent.com/sushiswap/list/master/lists/token-lists/default-token-list/tokens/xdai.json",
	},
	// base
	8453: {
		ERC20:   "https://raw.githubusercontent.com/0xsequence/token-directory/master/index/base/erc20.json",
		ERC721:  "https://raw.githubusercontent.com/0xsequence/token-directory/master/index/base/erc721.json",
		ERC1155: "https://raw.githubusercontent.com/0xsequence/token-directory/master/index/base/erc1155.json",
		Misc:    "https://raw.githubusercontent.com/0xsequence/token-directory/master/index/base/misc.json",
	},
	// base-goerli
	84531: {
		ERC20:   "https://raw.githubusercontent.com/0xsequence/token-directory/master/index/base-goerli/erc20.json",
		ERC721:  "https://raw.githubusercontent.com/0xsequence/token-directory/master/index/base-goerli/erc721.json",
		ERC1155: "https://raw.githubusercontent.com/0xsequence/token-directory/master/index/base-goerli/erc1155.json",
		Misc:    "https://raw.githubusercontent.com/0xsequence/token-directory/master/index/base-goerli/misc.json",
	},
	// homeverse
	19011: {
		ERC20:   "https://raw.githubusercontent.com/0xsequence/token-directory/master/index/homeverse/erc20.json",
		ERC721:  "https://raw.githubusercontent.com/0xsequence/token-directory/master/index/homeverse/erc721.json",
		ERC1155: "https://raw.githubusercontent.com/0xsequence/token-directory/master/index/homeverse/erc1155.json",
		Misc:    "https://raw.githubusercontent.com/0xsequence/token-directory/master/index/homeverse/misc.json",
	},
	// homeverse-testnet
	40875: {
		ERC20:   "https://raw.githubusercontent.com/0xsequence/token-directory/master/index/homeverse-testnet/erc20.json",
		ERC721:  "https://raw.githubusercontent.com/0xsequence/token-directory/master/index/homeverse-testnet/erc721.json",
		ERC1155: "https://raw.githubusercontent.com/0xsequence/token-directory/master/index/homeverse-testnet/erc1155.json",
		Misc:    "https://raw.githubusercontent.com/0xsequence/token-directory/master/index/homeverse-testnet/misc.json",
	},
	// sepolia
	11155111: {
		ERC20:   "https://raw.githubusercontent.com/0xsequence/token-directory/master/index/sepolia/erc20.json",
		ERC721:  "https://raw.githubusercontent.com/0xsequence/token-directory/master/index/sepolia/erc721.json",
		ERC1155: "https://raw.githubusercontent.com/0xsequence/token-directory/master/index/sepolia/erc1155.json",
		Misc:    "https://raw.githubusercontent.com/0xsequence/token-directory/master/index/sepolia/misc.json",
	},
	// base-sepolia
	84532: {
		ERC20:   "https://raw.githubusercontent.com/0xsequence/token-directory/master/index/base-sepolia/erc20.json",
		ERC721:  "https://raw.githubusercontent.com/0xsequence/token-directory/master/index/base-sepolia/erc721.json",
		ERC1155: "https://raw.githubusercontent.com/0xsequence/token-directory/master/index/base-sepolia/erc1155.json",
		Misc:    "https://raw.githubusercontent.com/0xsequence/token-directory/master/index/base-sepolia/misc.json",
	},
	// arbitrum-sepolia
	421614: {
		ERC20:   "https://raw.githubusercontent.com/0xsequence/token-directory/master/index/arbitrum-sepolia/erc20.json",
		ERC721:  "https://raw.githubusercontent.com/0xsequence/token-directory/master/index/arbitrum-sepolia/erc721.json",
		ERC1155: "https://raw.githubusercontent.com/0xsequence/token-directory/master/index/arbitrum-sepolia/erc1155.json",
		Misc:    "https://raw.githubusercontent.com/0xsequence/token-directory/master/index/arbitrum-sepolia/misc.json",
	},
	// optimism-sepolia
	11155420: {
		ERC20:   "https://raw.githubusercontent.com/0xsequence/token-directory/master/index/optimism-sepolia/erc20.json",
		ERC721:  "https://raw.githubusercontent.com/0xsequence/token-directory/master/index/optimism-sepolia/erc721.json",
		ERC1155: "https://raw.githubusercontent.com/0xsequence/token-directory/master/index/optimism-sepolia/erc1155.json",
		Misc:    "https://raw.githubusercontent.com/0xsequence/token-directory/master/index/optimism-sepolia/misc.json",
	},
}
