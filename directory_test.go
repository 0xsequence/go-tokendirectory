package gotokendirectory

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestImageURI(t *testing.T) {
	Sync()
	imageURIs := GetAllImageURI()
	assert.NotNil(t, imageURIs)
	fmt.Println(imageURIs)
}
