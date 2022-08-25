package gotokendirectory

var TokenDirectory []Directory

func GetAllImageURI() []string {
	var imageURIs []string
	for _, directory := range TokenDirectory {
		for _, token := range directory.Tokens {
			imageURIs = append(imageURIs, token.LogoURI)
		}
	}

	return imageURIs
}
