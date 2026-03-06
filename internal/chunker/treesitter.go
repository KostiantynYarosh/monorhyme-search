//go:build ignore

// Tree-sitter chunking requires CGO and explicit opt-in with the "tree_sitter"
// build tag. Build with: go build -tags tree_sitter .
// Also requires adding grammar deps to go.mod:
//   github.com/tree-sitter/go-tree-sitter
//   github.com/tree-sitter/tree-sitter-go/bindings/go
//   github.com/tree-sitter/tree-sitter-python
//   github.com/tree-sitter/tree-sitter-javascript
//   github.com/tree-sitter/tree-sitter-typescript
// and a C compiler (MinGW-w64 on Windows).

package chunker

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	sitter "github.com/tree-sitter/go-tree-sitter"
	golang "github.com/tree-sitter/tree-sitter-go/bindings/go"
	javascript "github.com/tree-sitter/tree-sitter-javascript/bindings/go"
	python "github.com/tree-sitter/tree-sitter-python/bindings/go"
	typescript "github.com/tree-sitter/tree-sitter-typescript/bindings/go"
)

// TreeSitterChunker uses tree-sitter to extract function/method-level chunks
// from supported source files. Requires CGO and the tree_sitter build tag.
type TreeSitterChunker struct{}

// NewTreeSitter creates a TreeSitterChunker.
func NewTreeSitter() *TreeSitterChunker { return &TreeSitterChunker{} }

var codeExtensions = map[string]bool{
	".go": true, ".py": true, ".js": true, ".jsx": true,
	".ts": true, ".tsx": true,
}

func (t *TreeSitterChunker) Supports(ext string) bool {
	return codeExtensions[strings.ToLower(ext)]
}

func (t *TreeSitterChunker) Chunk(path string) ([]Chunk, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if isBinary(data) {
		return nil, nil
	}

	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	modTime := info.ModTime()

	lang, queryStr, err := langForExt(filepath.Ext(path))
	if err != nil {
		return nil, err
	}

	parser := sitter.NewParser()
	defer parser.Close()
	parser.SetLanguage(lang)

	tree := parser.Parse(data, nil)
	defer tree.Close()

	q, err := sitter.NewQuery(lang, queryStr)
	if err != nil {
		return nil, fmt.Errorf("tree-sitter query: %w", err)
	}
	defer q.Close()

	cursor := sitter.NewQueryCursor()
	defer cursor.Close()
	matches := cursor.Matches(q, tree.RootNode(), data)

	var chunks []Chunk
	for match := matches.Next(); match != nil; match = matches.Next() {
		for _, capture := range match.Captures {
			node := capture.Node
			startLine := int(node.StartPosition().Row) + 1
			endLine := int(node.EndPosition().Row) + 1
			content := string(data[node.StartByte():node.EndByte()])
			chunks = append(chunks, Chunk{
				ID:        MakeID(path, startLine),
				FilePath:  path,
				StartLine: startLine,
				EndLine:   endLine,
				Content:   content,
				ModTime:   modTime,
			})
		}
	}

	if len(chunks) == 0 {
		sw := NewSlidingWindow(300, 50)
		return sw.Chunk(path)
	}
	return chunks, nil
}

func langForExt(ext string) (*sitter.Language, string, error) {
	const goQuery = `[(function_declaration) @chunk (method_declaration) @chunk]`
	const pyQuery = `[(function_definition) @chunk]`
	const jsQuery = `[(function_declaration) @chunk (method_definition) @chunk (arrow_function) @chunk]`
	const tsQuery = `[(function_declaration) @chunk (method_definition) @chunk (arrow_function) @chunk]`

	switch strings.ToLower(ext) {
	case ".go":
		return sitter.NewLanguage(golang.Language()), goQuery, nil
	case ".py":
		return sitter.NewLanguage(python.Language()), pyQuery, nil
	case ".js", ".jsx":
		return sitter.NewLanguage(javascript.Language()), jsQuery, nil
	case ".ts", ".tsx":
		return sitter.NewLanguage(typescript.LanguageTypescript()), tsQuery, nil
	default:
		return nil, "", fmt.Errorf("unsupported extension %q", ext)
	}
}
