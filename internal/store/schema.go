package store

const schema = `
CREATE TABLE IF NOT EXISTS chunks (
    id         TEXT    PRIMARY KEY,
    file_path  TEXT    NOT NULL,
    start_line INTEGER NOT NULL,
    end_line   INTEGER NOT NULL,
    content    TEXT    NOT NULL,
    embedding  BLOB    NOT NULL,
    mod_time   INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_chunks_file_path ON chunks(file_path);

CREATE TABLE IF NOT EXISTS files (
    path        TEXT    PRIMARY KEY,
    mod_time    INTEGER NOT NULL,
    chunk_count INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS meta (
    key   TEXT PRIMARY KEY,
    value TEXT NOT NULL
);
`
