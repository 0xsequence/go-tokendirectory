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
