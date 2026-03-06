package chunker

import (
	"path/filepath"
	"strings"
)

var defaultTreeSitter = NewTreeSitter()

func SelectChunker(path string, maxTokens, overlapTokens int) FileChunker {
	ext := strings.ToLower(filepath.Ext(path))
	if defaultTreeSitter.Supports(ext) {
		return defaultTreeSitter
	}
	return NewSlidingWindow(maxTokens, overlapTokens)
}
