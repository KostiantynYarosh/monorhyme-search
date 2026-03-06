package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/user/monorhyme-search/internal/config"
	"github.com/user/monorhyme-search/internal/embedder"
	"github.com/user/monorhyme-search/internal/indexer"
	"github.com/user/monorhyme-search/internal/store"
)

var indexCmd = &cobra.Command{
	Use:   "index [path]",
	Short: "Index a directory for semantic search",
	Long:  "Walk the directory, chunk files, generate embeddings, and store them.",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runIndex,
}

var indexIgnore []string

func init() {
	rootCmd.AddCommand(indexCmd)
	indexCmd.Flags().StringArrayVar(&indexIgnore, "ignore", nil, "skip files under this path (repeatable)")
}

func runIndex(cmd *cobra.Command, args []string) error {
	root := "."
	if len(args) > 0 {
		root = args[0]
	}

	absRoot, err := absPath(root)
	if err != nil {
		return err
	}

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	if err := os.MkdirAll(dirOf(cfg.DBPath), 0o755); err != nil {
		return fmt.Errorf("create data dir: %w", err)
	}

	st, err := store.Open(cfg.DBPath)
	if err != nil {
		return fmt.Errorf("open store: %w", err)
	}
	defer st.Close()

	emb, err := embedder.New(cfg)
	if err != nil {
		return err
	}

	idx := indexer.New(st, emb, cfg)
	if len(indexIgnore) > 0 {
		idx.SetIgnorePaths(indexIgnore)
	}
	return idx.IndexPath(absRoot)
}

func absPath(p string) (string, error) {
	if p == "" {
		return os.Getwd()
	}
	info, err := os.Stat(p)
	if err != nil {
		return "", fmt.Errorf("path %q: %w", p, err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("path %q is not a directory", p)
	}
	abs, err := os.Getwd()
	if err != nil {
		return p, nil
	}
	_ = abs
	return p, nil
}

func dirOf(path string) string {
	if path == "" {
		return "."
	}
	dir := ""
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' || path[i] == '\\' {
			dir = path[:i]
			break
		}
	}
	if dir == "" {
		return "."
	}
	return dir
}
