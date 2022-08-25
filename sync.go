package gotokendirectory

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

func Sync() {
	files, err := GetFiles()
	if err != nil {
		fmt.Println(err)
	}
	SyncDirectory(files)
}

func SyncDirectory(files []string) {
	for _, file := range files {
		jsonFile, err := os.Open(file)
		if err != nil {
			fmt.Println(err)
		}
		defer jsonFile.Close()

		byteValue, _ := ioutil.ReadAll(jsonFile)

		var result Directory

		json.Unmarshal([]byte(byteValue), &result)

		TokenDirectory = append(TokenDirectory, result)
	}
}

func GetFiles() ([]string, error) {
	var files []string
	err := filepath.WalkDir("./token-directory/index", func(path string, d fs.DirEntry, err error) error {
		if d.IsDir() {
			return nil
		}

		if strings.HasSuffix(path, ".json") {
			files = append(files, path)
			return nil
		}

		return nil
	})

	return files, err
}
