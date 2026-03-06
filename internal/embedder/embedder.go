package embedder

import (
	"github.com/user/monorhyme-search/internal/config"
)

type Pinger interface {
	Ping() error
}

type Embedder interface {
	Embed(text string) ([]float32, error)
	EmbedBatch(texts []string) ([][]float32, error)
	Dim() int
}

func New(cfg *config.Config) (Embedder, error) {
	return NewOllama(cfg.OllamaBaseURL, cfg.OllamaModel), nil
}
