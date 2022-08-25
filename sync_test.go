package gotokendirectory

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFiles(t *testing.T) {
	files, err := GetFiles()
	assert.NoError(t, err)
	assert.NotNil(t, files)
}

func TestSync(t *testing.T) {
	files, err := GetFiles()
	assert.NoError(t, err)
	assert.NotNil(t, files)
	SyncDirectory(files)
}
