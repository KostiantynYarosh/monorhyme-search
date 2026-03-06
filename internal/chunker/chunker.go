package chunker

import (
	"crypto/sha256"
	"fmt"
	"strconv"
	"time"
)

type Chunk struct {
	ID        string
	FilePath  string
	StartLine int
	EndLine   int
	Content   string
	Embedding []float32
	ModTime   time.Time
}

func MakeID(filePath string, startLine int) string {
	h := sha256.Sum256([]byte(filePath + ":" + strconv.Itoa(startLine)))
	return fmt.Sprintf("%x", h)
}

type FileChunker interface {
	Chunk(path string) ([]Chunk, error)
	Supports(ext string) bool
}
