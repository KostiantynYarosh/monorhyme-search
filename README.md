# monorhyme-search

Semantic search for local files. Indexes code and text using vector embeddings, then lets you search by meaning rather than exact keywords. Works fully offline via [Ollama](https://ollama.com).

## Requirements

- [Ollama](https://ollama.com/download) running locally
- A pulled embedding model (default: `bge-m3`)

```
ollama pull bge-m3
```

## Installation

```
go build -o monorhyme-search.exe .
```

Place the binary anywhere. `config.yaml` is automatically created next to the binary on first run with all default values. Run `monorhyme-search config` to customize.

## Quick Start

```
# 1. Configure (optional — defaults work out of the box)
./monorhyme-search config

# 2. Index a directory
./monorhyme-search index F:\Study\KPI

# 3. Search
./monorhyme-search search "symmetric encryption algorithm"
```

## Commands

### `monorhyme-search index [path]`

Walks the directory, chunks files, generates embeddings, and stores them in the index. Subsequent runs only re-process changed files (incremental by mtime).

```
./monorhyme-search index F:\Study\KPI
./monorhyme-search index F:\Study\KPI --ignore "F:\Study\KPI\year1"
./monorhyme-search index F:\Study\KPI --ignore "F:\Study\KPI\year1" --ignore "F:\Study\KPI\year2"
```

**Flags:**
| Flag | Description |
|------|-------------|
| `--ignore <path>` | Skip files under this path (repeatable) |

**Automatically skipped:**
- Binary files (`.exe`, `.dll`, `.zip`, `.pdf`, images, video, etc.)
- Files with no extension
- Directories: `.git`, `node_modules`, `vendor`, `dist`, `build`, `venv`, `checkpoints`, `.cache`, and all dot-directories
- Files larger than 100 MB

**Custom ignore rules:** Create `.monorhyme-search-ignore` in the indexed directory (gitignore syntax).

---

### `monorhyme-search search <query>`

Embeds the query and returns the most semantically similar files.

```
./monorhyme-search search "how to hash a password"
./monorhyme-search search "лабораторна робота з мережевої безпеки"
./monorhyme-search search "database schema" --top 10
./monorhyme-search search "encryption" --path "F:\Study\KPI\year4"
./monorhyme-search search "encryption" --ignore "F:\Study\KPI\year1"
./monorhyme-search search "query" --ext .go
./monorhyme-search search "query" --min-score 0.5
./monorhyme-search search "query" --json
```

**Flags:**
| Flag | Default | Description |
|------|---------|-------------|
| `--top <n>` | 4 | Number of results to show |
| `--min-score <f>` | 0.3 | Minimum similarity score (0–1) |
| `--path <string>` | | Only show results whose file path contains this string |
| `--ignore <path>` | | Exclude files under this path from results (repeatable) |
| `--ext <.ext>` | | Filter by file extension, e.g. `.go` |
| `--json` | | Output results as JSON |

**Output format:**
```
[0.82] F:\Study\KPI\year4\crypto\lab2.md:14
    Symmetric encryption uses the same key for both encryption and decryption...
```

---

### `monorhyme-search status`

Shows index statistics.

```
./monorhyme-search status
```

```
Index:        C:\Users\user\AppData\Local\monorhyme-search\index.db (24.3 MB)
Files:        312
Chunks:       9,847
Last indexed: 2026-03-06 14:22
```

---

### `monorhyme-search clear`

Deletes the entire index or removes entries for a specific path.

```
# Delete the entire index
./monorhyme-search clear

# Remove only files under a specific path
./monorhyme-search clear --path "F:\Study\KPI\year1"
```

**Flags:**
| Flag | Description |
|------|-------------|
| `--path <path>` | Remove only files whose path starts with this prefix |

After clearing a path, re-index just that directory without affecting the rest:
```
./monorhyme-search clear --path "F:\Study\KPI\year4"
./monorhyme-search index "F:\Study\KPI\year4"
```

---

### `monorhyme-search config`

Interactive setup. Prompts for settings and saves to `config.yaml` next to the binary.

```
./monorhyme-search config
```

```
monorhyme-search config
===============

Ollama base URL [http://localhost:11434]:
Ollama model (embedding model name) [bge-m3]:
SQLite database path [C:\Users\user\AppData\Local\monorhyme-search\index.db]:
Default number of search results [4]:
Chunks per embedding HTTP call (index_batch_size) [32]:
Sliding window size in tokens (chunk_max_tokens) [300]:
Overlap between chunks in tokens (chunk_overlap_tokens) [50]:

Config saved to: F:\projects\monorhyme-search\config.yaml
```

---

## Configuration

`config.yaml` is created next to the binary. Only non-default values need to be present.

```yaml
ollama_base_url: http://localhost:11434
ollama_model: bge-m3
db_path: C:\Users\user\AppData\Local\monorhyme-search\index.db
search_top_n: 4
index_batch_size: 32
chunk_max_tokens: 300
chunk_overlap_tokens: 50
```

**All options:**
| Key | Default | Description |
|-----|---------|-------------|
| `ollama_base_url` | `http://localhost:11434` | Ollama server URL |
| `ollama_model` | `bge-m3` | Embedding model name |
| `db_path` | `%LOCALAPPDATA%\monorhyme-search\index.db` | SQLite database path |
| `search_top_n` | `4` | Default number of search results |
| `index_batch_size` | `32` | Chunks per embedding HTTP call |
| `chunk_max_tokens` | `300` | Sliding window size (whitespace tokens) |
| `chunk_overlap_tokens` | `50` | Overlap between consecutive chunks |

---

## Data Storage

| File | Location |
|------|----------|
| Config | Next to the binary (`config.yaml`) |
| Index DB | `%LOCALAPPDATA%\monorhyme-search\index.db` (Windows) / `~/.cache/monorhyme-search/index.db` (Linux/macOS) |

The index persists across reboots. To start fresh: `monorhyme-search clear`.

---

## Changing the Model

If you switch the embedding model, the existing index becomes incompatible (different vector dimensions). monorhyme-search will detect this and show an error:

```
model mismatch: index was built with a 768-dim model, current model produces 1024-dim
Run: monorhyme-search clear && monorhyme-search index <path>
```

**Recommended models:**
| Model | Dimensions | Best for |
|-------|-----------|---------|
| `bge-m3` | 1024 | Multilingual (Ukrainian, Russian, English) |
| `nomic-embed-text` | 768 | English only, faster |
| `mxbai-embed-large` | 1024 | English, high quality |

---

## Global `--config` Flag

Override the config file location:

```
./monorhyme-search --config /path/to/custom.yaml search "query"
```
