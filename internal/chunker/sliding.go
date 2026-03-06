package chunker

import (
	"bytes"
	"os"
	"strings"
)

type SlidingWindowChunker struct {
	maxTokens     int
	overlapTokens int
}

func NewSlidingWindow(maxTokens, overlapTokens int) *SlidingWindowChunker {
	if maxTokens <= 0 {
		maxTokens = 300
	}
	if overlapTokens < 0 {
		overlapTokens = 0
	}
	return &SlidingWindowChunker{maxTokens: maxTokens, overlapTokens: overlapTokens}
}

func (s *SlidingWindowChunker) Supports(_ string) bool { return true }

type token struct {
	text string
	line int
}

func (s *SlidingWindowChunker) Chunk(path string) ([]Chunk, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if isBinary(data) {
		return nil, nil
	}

	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	modTime := info.ModTime()

	tokens := tokenize(data)
	if len(tokens) == 0 {
		return nil, nil
	}

	step := s.maxTokens - s.overlapTokens
	if step <= 0 {
		step = s.maxTokens
	}

	var chunks []Chunk
	for start := 0; start < len(tokens); start += step {
		end := start + s.maxTokens
		if end > len(tokens) {
			end = len(tokens)
		}

		window := tokens[start:end]
		startLine := window[0].line
		endLine := window[len(window)-1].line
		content := joinTokens(window)

		chunks = append(chunks, Chunk{
			ID:        MakeID(path, startLine),
			FilePath:  path,
			StartLine: startLine,
			EndLine:   endLine,
			Content:   content,
			ModTime:   modTime,
		})

		if end == len(tokens) {
			break
		}
	}
	return chunks, nil
}

func isWS(b byte) bool {
	return b == ' ' || b == '\t' || b == '\r' || b == '\n' || b == '\v' || b == '\f'
}

func tokenize(data []byte) []token {
	var tokens []token
	line := 1
	i := 0

	for i < len(data) {
		for i < len(data) && isWS(data[i]) {
			if data[i] == '\n' {
				line++
			}
			i++
		}
		if i >= len(data) {
			break
		}

		j := i
		for j < len(data) && !isWS(data[j]) {
			j++
		}
		tokens = append(tokens, token{text: string(data[i:j]), line: line})
		i = j
	}
	return tokens
}

func joinTokens(tokens []token) string {
	words := make([]string, len(tokens))
	for i, t := range tokens {
		words[i] = t.text
	}
	return strings.Join(words, " ")
}

func isBinary(data []byte) bool {
	sample := data
	if len(sample) > 512 {
		sample = sample[:512]
	}
	return bytes.IndexByte(sample, 0) >= 0
}
