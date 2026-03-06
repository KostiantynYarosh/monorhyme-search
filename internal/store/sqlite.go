package store

import (
	"database/sql"
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"time"

	"github.com/user/monorhyme-search/internal/chunker"
	_ "modernc.org/sqlite"
)

type SQLiteStore struct {
	db *sql.DB
}

func Open(path string) (*SQLiteStore, error) {
	if dir := filepath.Dir(path); dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("create db directory: %w", err)
		}
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite %q: %w", path, err)
	}

	pragmas := []string{
		"PRAGMA journal_mode=WAL",
		"PRAGMA synchronous=NORMAL",
		"PRAGMA foreign_keys=ON",
	}
	for _, p := range pragmas {
		if _, err := db.Exec(p); err != nil {
			db.Close()
			return nil, fmt.Errorf("pragma %q: %w", p, err)
		}
	}

	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("apply schema: %w", err)
	}

	return &SQLiteStore{db: db}, nil
}

func (s *SQLiteStore) Close() error {
	return s.db.Close()
}

func (s *SQLiteStore) SaveChunks(chunks []chunker.Chunk) error {
	if len(chunks) == 0 {
		return nil
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT OR REPLACE INTO chunks (id, file_path, start_line, end_line, content, embedding, mod_time)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, c := range chunks {
		blob, err := encodeEmbedding(c.Embedding)
		if err != nil {
			return fmt.Errorf("encode embedding for chunk %s: %w", c.ID, err)
		}
		if _, err := stmt.Exec(
			c.ID, c.FilePath, c.StartLine, c.EndLine, c.Content,
			blob, c.ModTime.UnixNano(),
		); err != nil {
			return fmt.Errorf("insert chunk %s: %w", c.ID, err)
		}
	}
	return tx.Commit()
}

func (s *SQLiteStore) GetChunks() ([]chunker.Chunk, error) {
	rows, err := s.db.Query(
		`SELECT id, file_path, start_line, end_line, content, embedding, mod_time FROM chunks`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var chunks []chunker.Chunk
	for rows.Next() {
		var c chunker.Chunk
		var blob []byte
		var nanos int64
		if err := rows.Scan(&c.ID, &c.FilePath, &c.StartLine, &c.EndLine, &c.Content, &blob, &nanos); err != nil {
			return nil, err
		}
		c.ModTime = time.Unix(0, nanos)
		c.Embedding, err = decodeEmbedding(blob)
		if err != nil {
			return nil, fmt.Errorf("decode embedding for chunk %s: %w", c.ID, err)
		}
		chunks = append(chunks, c)
	}
	return chunks, rows.Err()
}

func (s *SQLiteStore) DeleteChunksForFile(path string) error {
	_, err := s.db.Exec(`DELETE FROM chunks WHERE file_path = ?`, path)
	return err
}

func (s *SQLiteStore) SaveFileMeta(path string, modTime time.Time, chunkCount int) error {
	_, err := s.db.Exec(
		`INSERT OR REPLACE INTO files (path, mod_time, chunk_count) VALUES (?, ?, ?)`,
		path, modTime.UnixNano(), chunkCount,
	)
	return err
}

func (s *SQLiteStore) GetFileModTime(path string) (time.Time, bool, error) {
	var nanos int64
	err := s.db.QueryRow(`SELECT mod_time FROM files WHERE path = ?`, path).Scan(&nanos)
	if err == sql.ErrNoRows {
		return time.Time{}, false, nil
	}
	if err != nil {
		return time.Time{}, false, err
	}
	return time.Unix(0, nanos), true, nil
}

func (s *SQLiteStore) DeleteFileMeta(path string) error {
	_, err := s.db.Exec(`DELETE FROM files WHERE path = ?`, path)
	return err
}

func (s *SQLiteStore) GetIndexedPaths() ([]string, error) {
	rows, err := s.db.Query(`SELECT path FROM files`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var paths []string
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			return nil, err
		}
		paths = append(paths, p)
	}
	return paths, rows.Err()
}

func (s *SQLiteStore) GetStats() (Stats, error) {
	var stats Stats

	if err := s.db.QueryRow(`SELECT COUNT(*) FROM files`).Scan(&stats.FileCount); err != nil {
		return stats, err
	}
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM chunks`).Scan(&stats.ChunkCount); err != nil {
		return stats, err
	}

	var lastNanos sql.NullInt64
	s.db.QueryRow(`SELECT value FROM meta WHERE key = 'last_indexed'`).Scan(&lastNanos)
	if lastNanos.Valid {
		stats.LastIndexed = time.Unix(0, lastNanos.Int64)
	}

	if dbPath := s.dbPath(); dbPath != "" {
		if fi, err := os.Stat(dbPath); err == nil {
			stats.DBSizeBytes = fi.Size()
		}
	}
	return stats, nil
}

func (s *SQLiteStore) SetLastIndexed(t time.Time) error {
	_, err := s.db.Exec(
		`INSERT OR REPLACE INTO meta (key, value) VALUES ('last_indexed', ?)`,
		t.UnixNano(),
	)
	return err
}

func (s *SQLiteStore) GetEmbeddingDim() int {
	var blob []byte
	s.db.QueryRow(`SELECT embedding FROM chunks LIMIT 1`).Scan(&blob)
	return len(blob) / 4
}

func (s *SQLiteStore) dbPath() string {
	var path string
	s.db.QueryRow(`PRAGMA database_list`).Scan(nil, nil, &path)
	return path
}

func encodeEmbedding(v []float32) ([]byte, error) {
	b := make([]byte, len(v)*4)
	for i, f := range v {
		binary.LittleEndian.PutUint32(b[i*4:], math.Float32bits(f))
	}
	return b, nil
}

func decodeEmbedding(b []byte) ([]float32, error) {
	if len(b)%4 != 0 {
		return nil, fmt.Errorf("embedding blob length %d is not a multiple of 4", len(b))
	}
	v := make([]float32, len(b)/4)
	for i := range v {
		bits := binary.LittleEndian.Uint32(b[i*4:])
		v[i] = math.Float32frombits(bits)
	}
	return v, nil
}
