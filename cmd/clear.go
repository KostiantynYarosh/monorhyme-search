package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/user/monorhyme-search/internal/config"
	"github.com/user/monorhyme-search/internal/store"
)

var clearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Delete the search index (or entries for a specific path)",
	RunE:  runClear,
}

var clearPath string

func init() {
	clearCmd.Flags().StringVar(&clearPath, "path", "", "remove only entries whose path contains this substring")
	rootCmd.AddCommand(clearCmd)
}

func runClear(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	if clearPath == "" {
		if _, err := os.Stat(cfg.DBPath); os.IsNotExist(err) {
			fmt.Println("No index found.")
			return nil
		}
		if err := os.Remove(cfg.DBPath); err != nil {
			return fmt.Errorf("delete index: %w", err)
		}
		fmt.Printf("Index deleted: %s\n", cfg.DBPath)
		return nil
	}

	if _, err := os.Stat(cfg.DBPath); os.IsNotExist(err) {
		fmt.Println("No index found.")
		return nil
	}

	st, err := store.Open(cfg.DBPath)
	if err != nil {
		return fmt.Errorf("open store: %w", err)
	}
	defer st.Close()

	paths, err := st.GetIndexedPaths()
	if err != nil {
		return fmt.Errorf("list indexed paths: %w", err)
	}

	filterNorm := strings.ToLower(strings.ReplaceAll(clearPath, "\\", "/"))
	if !strings.HasSuffix(filterNorm, "/") {
		filterNorm += "/"
	}
	removed := 0
	for _, p := range paths {
		pNorm := strings.ToLower(strings.ReplaceAll(p, "\\", "/"))
		if pNorm == strings.TrimSuffix(filterNorm, "/") || strings.HasPrefix(pNorm, filterNorm) {
			if err := st.DeleteChunksForFile(p); err != nil {
				return fmt.Errorf("delete chunks for %s: %w", p, err)
			}
			if err := st.DeleteFileMeta(p); err != nil {
				return fmt.Errorf("delete meta for %s: %w", p, err)
			}
			removed++
		}
	}

	if removed == 0 {
		fmt.Printf("No indexed files under %q\n", clearPath)
	} else {
		fmt.Printf("Removed %d file(s) under %q from index\n", removed, clearPath)
	}
	return nil
}
