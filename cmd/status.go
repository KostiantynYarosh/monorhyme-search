package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/user/monorhyme-search/internal/config"
	"github.com/user/monorhyme-search/internal/store"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show index statistics",
	RunE:  runStatus,
}

func init() {
	rootCmd.AddCommand(statusCmd)
}

func runStatus(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	st, err := store.Open(cfg.DBPath)
	if err != nil {
		return fmt.Errorf("open store: %w", err)
	}
	defer st.Close()

	stats, err := st.GetStats()
	if err != nil {
		return fmt.Errorf("get stats: %w", err)
	}

	fmt.Printf("Index:        %s (%s)\n", cfg.DBPath, humanBytes(stats.DBSizeBytes))
	fmt.Printf("Files:        %d\n", stats.FileCount)
	fmt.Printf("Chunks:       %d\n", stats.ChunkCount)
	if !stats.LastIndexed.IsZero() {
		fmt.Printf("Last indexed: %s\n", stats.LastIndexed.Format("2006-01-02 15:04"))
	} else {
		fmt.Printf("Last indexed: never\n")
	}
	return nil
}

func humanBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}
