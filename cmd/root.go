package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/user/monorhyme-search/internal/config"
)

var cfgFile string

var rootCmd = &cobra.Command{
	Use:   "monorhyme-search",
	Short: "Semantic search for your local files",
	Long: `monorhyme-search indexes your code and notes and lets you search by meaning,
not by exact keyword. Works fully offline using Ollama.`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default: ~/.config/monorhyme-search/config.yaml)")
}

func initConfig() {
	config.SetDefaults()

	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		configDir, err := config.ConfigDir()
		if err == nil {
			viper.AddConfigPath(configDir)
		}
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")
	}

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			fmt.Fprintf(os.Stderr, "Error reading config: %v\n", err)
		}
	}
}
