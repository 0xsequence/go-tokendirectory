package tokendirectory

import (
	"time"
)

// TokenList structure based on https://raw.githubusercontent.com/Uniswap/token-lists/main/test/schema/example.tokenlist.json
type TokenList struct {
	Name          string         `json:"name"`
	ChainID       uint64         `json:"chainId"`
	TokenStandard string         `json:"tokenStandard"`
	LogoURI       string         `json:"logoURI"`
	Keywords      []string       `json:"keywords"`
	Timestamp     *time.Time     `json:"timestamp"`
	Tokens        []ContractInfo `json:"tokens"`
	Version       interface{}    `json:"version"`
}

type ContractInfo struct {
	ChainID     uint64                `json:"chainId"`
	Address     string                `json:"address"`
	Name        string                `json:"name"`
	Type        string                `json:"type"`
	Symbol      string                `json:"symbol,omitempty"`
	Decimals    uint64                `json:"decimals"`
	LogoURI     string                `json:"logoURI,omitempty"`
	Extensions  ContractInfoExtension `json:"extensions"`
	ContentHash uint64                `json:"-"`
}

type ContractInfoExtension struct {
	Link                  string   `json:"link,omitempty"`
	Description           string   `json:"description,omitempty"`
	OgImage               string   `json:"ogImage,omitempty"`
	OriginChainID         uint64   `json:"originChainId,omitempty"`
	OriginAddress         string   `json:"originAddress,omitempty"`
	Blacklist             bool     `json:"blacklist,omitempty"`
	ContractABIExtensions []string `json:"contractABIExtensions,omitempty"`
	Featured              int32    `json:"featured,omitempty"`
	Mute                  bool     `json:"mute,omitempty"`
	Verified              bool     `json:"verified"`
	VerifiedBy            string   `json:"verifiedBy,omitempty"`
}
