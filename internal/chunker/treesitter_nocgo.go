//go:build !tree_sitter

package chunker

type TreeSitterChunker struct{}

func NewTreeSitter() *TreeSitterChunker { return &TreeSitterChunker{} }

func (t *TreeSitterChunker) Supports(_ string) bool { return false }

func (t *TreeSitterChunker) Chunk(path string) ([]Chunk, error) { return nil, nil }
