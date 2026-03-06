package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/user/monorhyme-search/internal/config"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Interactive configuration setup",
	RunE:  runConfig,
}

func init() {
	rootCmd.AddCommand(configCmd)
}

func runConfig(cmd *cobra.Command, args []string) error {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("monorhyme-search config")
	fmt.Println("===============")
	fmt.Println()

	url := prompt(reader, "Ollama base URL", viper.GetString("ollama_base_url"))
	model := prompt(reader, "Ollama model (embedding model name)", viper.GetString("ollama_model"))
	dbPath := prompt(reader, "SQLite database path", viper.GetString("db_path"))
	topN := prompt(reader, "Default number of search results", fmt.Sprintf("%d", viper.GetInt("search_top_n")))
	batchSize := prompt(reader, "Chunks per embedding HTTP call (index_batch_size)", fmt.Sprintf("%d", viper.GetInt("index_batch_size")))
	chunkMax := prompt(reader, "Sliding window size in tokens (chunk_max_tokens)", fmt.Sprintf("%d", viper.GetInt("chunk_max_tokens")))
	chunkOverlap := prompt(reader, "Overlap between chunks in tokens (chunk_overlap_tokens)", fmt.Sprintf("%d", viper.GetInt("chunk_overlap_tokens")))

	viper.Set("ollama_base_url", url)
	viper.Set("ollama_model", model)
	viper.Set("db_path", dbPath)
	viper.Set("search_top_n", topN)
	viper.Set("index_batch_size", batchSize)
	viper.Set("chunk_max_tokens", chunkMax)
	viper.Set("chunk_overlap_tokens", chunkOverlap)

	configDir, err := config.ConfigDir()
	if err != nil {
		return fmt.Errorf("resolve config dir: %w", err)
	}
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	configFile := filepath.Join(configDir, "config.yaml")
	if err := viper.WriteConfigAs(configFile); err != nil {
		return fmt.Errorf("write config: %w", err)
	}

	fmt.Printf("\nConfig saved to: %s\n", configFile)
	return nil
}

func prompt(r *bufio.Reader, label, defaultVal string) string {
	fmt.Printf("%s [%s]: ", label, defaultVal)
	line, _ := r.ReadString('\n')
	line = strings.TrimSpace(line)
	if line == "" {
		return defaultVal
	}
	return line
}
