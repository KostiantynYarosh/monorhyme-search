package store

import (
	"time"

	"github.com/user/monorhyme-search/internal/chunker"
)

type Stats struct {
	FileCount   int
	ChunkCount  int
	LastIndexed time.Time
	DBSizeBytes int64
}

type Store interface {
	SaveChunks(chunks []chunker.Chunk) error
	GetChunks() ([]chunker.Chunk, error)
	DeleteChunksForFile(path string) error
	SaveFileMeta(path string, modTime time.Time, chunkCount int) error
	GetFileModTime(path string) (time.Time, bool, error)
	DeleteFileMeta(path string) error
	GetIndexedPaths() ([]string, error)
	GetStats() (Stats, error)
	Close() error
}
