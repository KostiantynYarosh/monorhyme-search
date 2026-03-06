package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/user/monorhyme-search/internal/config"
	"github.com/user/monorhyme-search/internal/embedder"
	"github.com/user/monorhyme-search/internal/search"
	"github.com/user/monorhyme-search/internal/store"
)

var (
	searchTop      int
	searchExt      string
	searchPath     string
	searchJSON     bool
	searchMinScore float64
	searchIgnore   []string
)

var searchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search indexed files by meaning",
	Long:  "Embed the query and return the most semantically similar chunks.",
	Args:  cobra.MinimumNArgs(1),
	RunE:  runSearch,
}

func init() {
	rootCmd.AddCommand(searchCmd)
	searchCmd.Flags().IntVar(&searchTop, "top", 0, "number of results (default from config)")
	searchCmd.Flags().StringVar(&searchExt, "ext", "", "filter by file extension, e.g. .go")
	searchCmd.Flags().BoolVar(&searchJSON, "json", false, "output results as JSON")
	searchCmd.Flags().Float64Var(&searchMinScore, "min-score", 0.3, "minimum similarity score to show (0–1)")
	searchCmd.Flags().StringVar(&searchPath, "path", "", "filter results to files whose path contains this string")
	searchCmd.Flags().StringArrayVar(&searchIgnore, "ignore", nil, "exclude files under this path (repeatable)")
}

func runSearch(cmd *cobra.Command, args []string) error {
	query := strings.Join(args, " ")

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	topN := cfg.SearchTopN
	if searchTop > 0 {
		topN = searchTop
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

	queryVec, err := emb.Embed(query)
	if err != nil {
		return fmt.Errorf("embed query: %w", err)
	}

	if storedDim := st.GetEmbeddingDim(); storedDim > 0 && storedDim != len(queryVec) {
		return fmt.Errorf("model mismatch: index was built with a %d-dim model, current model produces %d-dim\nRun: rm index.db && monorhyme-search index <path>", storedDim, len(queryVec))
	}

	searcher := search.NewSearcher(st)
	results, err := searcher.Search(queryVec, topN, searchExt, searchPath, searchIgnore)
	if err != nil {
		return fmt.Errorf("search: %w", err)
	}

	if len(results) == 0 {
		fmt.Println("No results found. Run `monorhyme-search index <path>` first.")
		return nil
	}

	if searchJSON {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(results)
	}

	for _, r := range results {
		if float64(r.Score) < searchMinScore {
			break
		}
		snippet := firstLine(r.Chunk.Content)
		fmt.Printf("[%.2f] %s:%d\n    %s\n", r.Score, r.Chunk.FilePath, r.Chunk.StartLine, snippet)
	}
	return nil
}

func firstLine(s string) string {
	s = strings.TrimSpace(s)
	idx := strings.IndexByte(s, '\n')
	if idx == -1 {
		if len(s) > 120 {
			return s[:120]
		}
		return s
	}
	line := s[:idx]
	if len(line) > 120 {
		return line[:120]
	}
	return line
}
