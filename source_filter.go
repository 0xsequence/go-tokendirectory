package tokendirectory

import (
	"path"
	"strings"
)

type SourceFilter []string

// Filter will filter the sources by the given filter list. A filter
// is a list of a full url, or the url base path. For example:
// for url https://example.com/tokens.json the base path is tokens.json
func (s SourceFilter) Filter(input Sources) Sources {
	if len(s) == 0 {
		return input
	}
	output := make(Sources, len(input))
	for chainID, sourceList := range input {
		list := make([]string, 0, len(sourceList))
		for _, src := range sourceList {
			if !s.accept(src) {
				continue
			}
			list = append(list, src)
		}
		if len(list) == 0 {
			continue
		}
		output[chainID] = list
	}
	return output
}

func (s SourceFilter) accept(src string) bool {
	name := path.Base(src)
	for _, f := range s {
		if src == f {
			return true
		}
		if !strings.HasPrefix(name, f) {
			continue
		}
		return true
	}
	return false
}
