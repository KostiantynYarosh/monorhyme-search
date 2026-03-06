package config

import (
	"fmt"

	"github.com/spf13/viper"
)

type Config struct {
	OllamaBaseURL      string `mapstructure:"ollama_base_url"`
	OllamaModel        string `mapstructure:"ollama_model"`
	DBPath             string `mapstructure:"db_path"`
	SearchTopN         int    `mapstructure:"search_top_n"`
	IndexBatchSize     int    `mapstructure:"index_batch_size"`
	ChunkMaxTokens     int    `mapstructure:"chunk_max_tokens"`
	ChunkOverlapTokens int    `mapstructure:"chunk_overlap_tokens"`
}

func SetDefaults() {
	dbPath, _ := DefaultDBPath()

	viper.SetDefault("ollama_base_url", "http://localhost:11434")
	viper.SetDefault("ollama_model", "nomic-embed-text")
	viper.SetDefault("db_path", dbPath)
	viper.SetDefault("search_top_n", 4)
	viper.SetDefault("index_batch_size", 32)
	viper.SetDefault("chunk_max_tokens", 300)
	viper.SetDefault("chunk_overlap_tokens", 50)
}

func Load() (*Config, error) {
	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}
	return &cfg, nil
}
