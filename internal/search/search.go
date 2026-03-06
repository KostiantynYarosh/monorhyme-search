package search

import (
	"math"
	"sort"
	"strings"

	"github.com/user/monorhyme-search/internal/chunker"
	"github.com/user/monorhyme-search/internal/store"
)

type SearchResult struct {
	Chunk chunker.Chunk `json:"chunk"`
	Score float32       `json:"score"`
}

type Searcher struct {
	store store.Store
}

func NewSearcher(s store.Store) *Searcher {
	return &Searcher{store: s}
}

func (s *Searcher) Search(queryEmbedding []float32, topN int, extFilter, pathFilter string, ignorePaths []string) ([]SearchResult, error) {
	chunks, err := s.store.GetChunks()
	if err != nil {
		return nil, err
	}

	pathFilterLower := strings.ToLower(pathFilter)

	normIgnore := make([]string, len(ignorePaths))
	for i, p := range ignorePaths {
		n := strings.ToLower(strings.ReplaceAll(p, "\\", "/"))
		if !strings.HasSuffix(n, "/") {
			n += "/"
		}
		normIgnore[i] = n
	}

	results := make([]SearchResult, 0, len(chunks))
	for _, c := range chunks {
		if extFilter != "" && !strings.HasSuffix(strings.ToLower(c.FilePath), strings.ToLower(extFilter)) {
			continue
		}
		if pathFilter != "" && !strings.Contains(strings.ToLower(c.FilePath), pathFilterLower) {
			continue
		}
		pNorm := strings.ToLower(strings.ReplaceAll(c.FilePath, "\\", "/"))
		ignored := false
		for _, ig := range normIgnore {
			if pNorm == strings.TrimSuffix(ig, "/") || strings.HasPrefix(pNorm, ig) {
				ignored = true
				break
			}
		}
		if ignored {
			continue
		}
		if len(c.Embedding) != len(queryEmbedding) {
			continue
		}
		score := CosineSimilarity(queryEmbedding, c.Embedding)
		results = append(results, SearchResult{Chunk: c, Score: score})
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	seen := make(map[string]bool)
	deduped := results[:0]
	for _, r := range results {
		if seen[r.Chunk.FilePath] {
			continue
		}
		seen[r.Chunk.FilePath] = true
		deduped = append(deduped, r)
	}
	results = deduped

	if topN > 0 && len(results) > topN {
		results = results[:topN]
	}
	return results, nil
}

func CosineSimilarity(a, b []float32) float32 {
	var dot, normA, normB float64
	for i := range a {
		ai, bi := float64(a[i]), float64(b[i])
		dot += ai * bi
		normA += ai * ai
		normB += bi * bi
	}
	if normA == 0 || normB == 0 {
		return 0
	}
	return float32(dot / (math.Sqrt(normA) * math.Sqrt(normB)))
}
