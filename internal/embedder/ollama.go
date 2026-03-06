package embedder

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

type OllamaEmbedder struct {
	baseURL string
	model   string
	client  *http.Client
}

func NewOllama(baseURL, model string) *OllamaEmbedder {
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}
	if model == "" {
		model = "nomic-embed-text"
	}
	return &OllamaEmbedder{
		baseURL: strings.TrimRight(baseURL, "/"),
		model:   model,
		client:  &http.Client{Timeout: 10 * time.Minute},
	}
}

func (o *OllamaEmbedder) Dim() int { return 768 }

func (o *OllamaEmbedder) Ping() error {
	quick := &http.Client{Timeout: 5 * time.Second}
	resp, err := quick.Get(o.baseURL + "/api/tags")
	if err != nil {
		return fmt.Errorf("cannot reach Ollama at %s\n  → Install: https://ollama.com/download\n  → Then run: ollama pull %s", o.baseURL, o.model)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Ollama returned HTTP %d at %s", resp.StatusCode, o.baseURL)
	}

	fmt.Fprintf(os.Stderr, "Loading model %q into memory", o.model)
	done := make(chan struct{})
	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				fmt.Fprintf(os.Stderr, ".")
			case <-done:
				return
			}
		}
	}()
	_, warmupErr := o.EmbedBatch([]string{"warmup"})
	close(done)
	fmt.Fprintln(os.Stderr, " ok")

	if warmupErr != nil {
		return fmt.Errorf("model warmup failed: %w\n  → Is the model pulled? Try: ollama pull %s", warmupErr, o.model)
	}
	return nil
}

func (o *OllamaEmbedder) Embed(text string) ([]float32, error) {
	vecs, err := o.EmbedBatch([]string{text})
	if err != nil {
		return nil, err
	}
	return vecs[0], nil
}

type ollamaEmbedRequest struct {
	Model string   `json:"model"`
	Input []string `json:"input"`
}

type ollamaEmbedResponse struct {
	Embeddings [][]float32 `json:"embeddings"`
}

func (o *OllamaEmbedder) EmbedBatch(texts []string) ([][]float32, error) {
	payload, err := json.Marshal(ollamaEmbedRequest{Model: o.model, Input: texts})
	if err != nil {
		return nil, err
	}

	resp, err := o.client.Post(o.baseURL+"/api/embed", "application/json", bytes.NewReader(payload))
	if err != nil {
		if strings.Contains(err.Error(), "connection refused") {
			return nil, fmt.Errorf("cannot connect to Ollama at %s — is it running? Try: ollama serve", o.baseURL)
		}
		return nil, fmt.Errorf("ollama request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ollama returned HTTP %d — is model %q pulled? Try: ollama pull %s", resp.StatusCode, o.model, o.model)
	}

	var result ollamaEmbedResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode ollama response: %w", err)
	}
	if len(result.Embeddings) != len(texts) {
		return nil, fmt.Errorf("ollama returned %d embeddings for %d inputs", len(result.Embeddings), len(texts))
	}
	return result.Embeddings, nil
}
