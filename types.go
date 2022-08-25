package gotokendirectory

type Directory struct {
	Name          string   `json:"name"`
	ChainId       int      `json:"chainId"`
	TokenStandard string   `json:"tokenStandard"`
	LogoURI       string   `json:"logoURI"`
	Tokens        []Tokens `json:"tokens"`
}

type Tokens struct {
	Name       string         `json:"name"`
	Symbol     string         `json:"symbol"`
	Decimals   int            `json:"decimals"`
	ChainId    int            `json:"chainId"`
	Address    string         `json:"address"`
	LogoURI    string         `json:"logoURI"`
	Extensions TokenExtension `json:"extensions"`
}

type TokenExtension struct {
	Link          string `json:"link"`
	Descriptions  string `json:"descriptions"`
	OgImage       string `json:"ogImage"`
	OriginChainId int    `json:"originChainId"`
	OriginAddress string `json:"originAddress"`
}
