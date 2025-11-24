package tokendirectory

import "time"

// TokenList structure based on https://raw.githubusercontent.com/Uniswap/token-lists/main/test/schema/example.tokenlist.json
type TokenList struct {
	Name          string         `json:"name"`
	ChainID       uint64         `json:"chainId"`
	TokenStandard string         `json:"tokenStandard"` // added by 0xsequence/token-directory
	LogoURI       string         `json:"logoURI"`
	Keywords      []string       `json:"keywords"`
	Timestamp     *time.Time     `json:"timestamp"`
	Tokens        []ContractInfo `json:"tokens"`
	Version       interface{}    `json:"version"`

	// Fields for internal use as represented in the master index.json
	TokenListURL string `json:"-"`
	ContentHash  string `json:"-"`
	Deprecated   bool   `json:"-"`
}

type ContractInfo struct {
	ChainID     uint64                `json:"chainId"`
	Address     string                `json:"address"`
	Name        string                `json:"name"`
	Type        string                `json:"type,omitempty"` // added by 0xsequence/token-directory
	Symbol      string                `json:"symbol,omitempty"`
	Decimals    *uint64               `json:"decimals"`
	LogoURI     string                `json:"logoURI,omitempty"`
	Extensions  ContractInfoExtension `json:"extensions"`
	ContentHash uint64                `json:"-"`
}

type ContractInfoExtension struct {
	Link        string   `json:"link,omitempty"`
	Description string   `json:"description,omitempty"`
	Categories  []string `json:"categories,omitempty"`
	BridgeInfo  map[string]struct {
		TokenAddress string `json:"tokenAddress"`
	} `json:"bridgeInfo,omitempty"`
	IndexingInfo map[string]struct {
		UseOnChainBalance bool `json:"useOnChainBalance"`
	} `json:"indexingInfo,omitempty"`

	OgName        string `json:"ogName,omitempty"`
	OgImage       string `json:"ogImage,omitempty"`
	OriginChainID uint64 `json:"originChainId,omitempty"`
	OriginAddress string `json:"originAddress,omitempty"`

	Blacklist bool `json:"blacklist,omitempty"`
	Mute      bool `json:"mute,omitempty"`

	SupportsDecimals bool `json:"supportsDecimals,omitempty"`

	Featured     bool `json:"featured,omitempty"`
	FeatureIndex int  `json:"featureIndex,omitempty"`

	Verified   bool   `json:"verified"`
	VerifiedBy string `json:"verifiedBy,omitempty"`
}
