package indexer

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/schollz/progressbar/v3"
	gitignore "github.com/sabhiram/go-gitignore"
	"github.com/user/monorhyme-search/internal/chunker"
	"github.com/user/monorhyme-search/internal/config"
	"github.com/user/monorhyme-search/internal/embedder"
	"github.com/user/monorhyme-search/internal/store"
)

type Indexer struct {
	store       store.Store
	embedder    embedder.Embedder
	cfg         *config.Config
	ignorePaths []string
}

func New(s store.Store, e embedder.Embedder, cfg *config.Config) *Indexer {
	return &Indexer{store: s, embedder: e, cfg: cfg}
}

func (idx *Indexer) SetIgnorePaths(paths []string) {
	norm := make([]string, len(paths))
	for i, p := range paths {
		n := strings.ToLower(strings.ReplaceAll(p, "\\", "/"))
		if !strings.HasSuffix(n, "/") {
			n += "/"
		}
		norm[i] = n
	}
	idx.ignorePaths = norm
}

func (idx *Indexer) IndexPath(root string) error {
	if p, ok := idx.embedder.(embedder.Pinger); ok {
		fmt.Fprintf(os.Stderr, "Checking embedder connection... ")
		if err := p.Ping(); err != nil {
			fmt.Fprintln(os.Stderr, "failed")
			return err
		}
		fmt.Fprintln(os.Stderr, "ok")
	}

	indexedPaths, err := idx.store.GetIndexedPaths()
	if err != nil {
		return fmt.Errorf("get indexed paths: %w", err)
	}
	visited := make(map[string]bool, len(indexedPaths))
	for _, p := range indexedPaths {
		visited[p] = false
	}

	var ignore *gitignore.GitIgnore
	ignorePath := filepath.Join(root, ".monorhyme-search-ignore")
	if _, err := os.Stat(ignorePath); err == nil {
		ignore, _ = gitignore.CompileIgnoreFile(ignorePath)
	}

	var toIndex []fileEntry
	var skipped int

	fmt.Fprintf(os.Stderr, "Scanning %s...\n", root)
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		name := d.Name()
		if d.IsDir() {
			if shouldSkipDir(name) || idx.isIgnored(path) {
				return filepath.SkipDir
			}
			return nil
		}
		if ignore != nil {
			rel, _ := filepath.Rel(root, path)
			if ignore.MatchesPath(rel) {
				return nil
			}
		}
		if idx.isIgnored(path) {
			return nil
		}
		if shouldSkipFile(path) {
			return nil
		}
		visited[path] = true

		info, err := d.Info()
		if err != nil {
			return nil
		}
		if info.Size() > 100*1024*1024 {
			return nil
		}
		storedMod, found, err := idx.store.GetFileModTime(path)
		if err != nil {
			return err
		}
		if found && storedMod.Equal(info.ModTime().Truncate(time.Second)) {
			skipped++
			return nil
		}
		toIndex = append(toIndex, fileEntry{path: path, info: info, isNew: !found})
		return nil
	})
	if err != nil {
		return fmt.Errorf("walk %s: %w", root, err)
	}

	fmt.Fprintf(os.Stderr, "Found %d file(s) to index, %d unchanged.\n", len(toIndex), skipped)

	if len(toIndex) > 0 {
		if err := idx.processFiles(root, toIndex); err != nil {
			return err
		}
	}

	pruned := 0
	for path, seen := range visited {
		if !seen {
			idx.store.DeleteChunksForFile(path)
			idx.store.DeleteFileMeta(path)
			pruned++
		}
	}

	if st, ok := idx.store.(*store.SQLiteStore); ok {
		st.SetLastIndexed(time.Now())
	}

	fmt.Fprintf(os.Stderr, "Done: %d indexed, %d skipped (unchanged), %d pruned (deleted)\n", len(toIndex), skipped, pruned)
	return nil
}

type fileEntry struct {
	path  string
	info  fs.FileInfo
	isNew bool
}

func (idx *Indexer) processFiles(root string, toIndex []fileEntry) error {
	type fileChunks struct {
		entry  fileEntry
		chunks []chunker.Chunk
	}

	fmt.Fprintf(os.Stderr, "Chunking %d file(s)...\n", len(toIndex))
	var all []fileChunks
	totalChunks := 0
	for _, f := range toIndex {
		chunks, err := chunker.SelectChunker(f.path, idx.cfg.ChunkMaxTokens, idx.cfg.ChunkOverlapTokens).Chunk(f.path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  warning: chunk %s: %v\n", f.path, err)
			continue
		}
		if len(chunks) == 0 {
			continue
		}
		all = append(all, fileChunks{entry: f, chunks: chunks})
		totalChunks += len(chunks)
		fmt.Fprintf(os.Stderr, "  %s — %d chunks\n", shortPath(root, f.path), len(chunks))
	}

	if totalChunks == 0 {
		return nil
	}

	fmt.Fprintf(os.Stderr, "Embedding %d chunks...\n", totalChunks)
	bar := progressbar.NewOptions(totalChunks,
		progressbar.OptionSetDescription("Embedding"),
		progressbar.OptionSetWriter(os.Stderr),
		progressbar.OptionShowCount(),
		progressbar.OptionShowIts(),
		progressbar.OptionSetItsString("chunks"),
		progressbar.OptionOnCompletion(func() { fmt.Fprintln(os.Stderr) }),
	)

	for _, fc := range all {
		f := fc.entry
		if !f.isNew {
			if err := idx.store.DeleteChunksForFile(f.path); err != nil {
				return fmt.Errorf("delete old chunks for %s: %w", f.path, err)
			}
		}

		if err := idx.embedChunksWithBar(fc.chunks, bar); err != nil {
			fmt.Fprintf(os.Stderr, "\nwarning: embed %s: %v\n", f.path, err)
			continue
		}

		if err := idx.store.SaveChunks(fc.chunks); err != nil {
			return fmt.Errorf("save chunks for %s: %w", f.path, err)
		}
		if err := idx.store.SaveFileMeta(f.path, f.info.ModTime().Truncate(time.Second), len(fc.chunks)); err != nil {
			return fmt.Errorf("save file meta for %s: %w", f.path, err)
		}
	}
	return nil
}

func (idx *Indexer) embedChunksWithBar(chunks []chunker.Chunk, bar *progressbar.ProgressBar) error {
	batchSize := idx.cfg.IndexBatchSize
	if batchSize <= 0 {
		batchSize = 8
	}

	for start := 0; start < len(chunks); start += batchSize {
		end := start + batchSize
		if end > len(chunks) {
			end = len(chunks)
		}
		batch := chunks[start:end]

		texts := make([]string, len(batch))
		for i, c := range batch {
			text := c.Content
			if len(text) > 4000 {
				text = text[:4000]
			}
			texts[i] = text
		}

		vecs, err := idx.embedder.EmbedBatch(texts)
		if err != nil {
			return err
		}

		for i := range batch {
			chunks[start+i].Embedding = vecs[i]
		}
		bar.Add(len(batch))
	}
	return nil
}

func (idx *Indexer) isIgnored(path string) bool {
	if len(idx.ignorePaths) == 0 {
		return false
	}
	pNorm := strings.ToLower(strings.ReplaceAll(path, "\\", "/"))
	for _, ig := range idx.ignorePaths {
		if pNorm == strings.TrimSuffix(ig, "/") || strings.HasPrefix(pNorm, ig) {
			return true
		}
	}
	return false
}

func shouldSkipDir(name string) bool {
	switch name {
	case ".git", ".hg", ".svn",
		"node_modules", "vendor",
		"dist", "build", "target", "out",
		".next", ".nuxt", "__pycache__",
		".venv", "venv", "env",
		"checkpoints", "checkpoint", ".cache":
		return true
	}
	return strings.HasPrefix(name, ".")
}

var skipFileExts = map[string]bool{
	".exe": true, ".dll": true, ".so": true, ".dylib": true,
	".a": true, ".o": true, ".obj": true, ".lib": true,
	".png": true, ".jpg": true, ".jpeg": true, ".gif": true,
	".bmp": true, ".ico": true, ".svg": true, ".webp": true,
	".tiff": true, ".tif": true, ".raw": true, ".cr2": true, ".nef": true,
	".heic": true, ".heif": true, ".avif": true, ".psd": true,
	".tga": true, ".dds": true, ".hdr": true, ".exr": true,
	".mp4": true, ".mp3": true, ".wav": true, ".avi": true,
	".mov": true, ".mkv": true, ".pdf": true, ".zip": true,
	".tar": true, ".gz": true, ".7z": true, ".rar": true,
	".wasm": true, ".pyc": true, ".class": true, ".dex": true,
	".csv": true, ".tsv": true, ".s": true, ".asm": true,
	".bin": true, ".com": true, ".msi": true, ".apk": true, ".elf": true,
	".vmdk": true, ".vhd": true, ".vhdx": true, ".ova": true, ".ovf": true, ".iso": true,
	".smali": true, ".odex": true, ".oat": true, ".art": true,
}

func shouldSkipFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	if ext == "" {
		return true
	}
	return skipFileExts[ext]
}

func shortPath(root, path string) string {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return filepath.Base(path)
	}
	return rel
}
