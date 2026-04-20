-- specd SQLite schema v1

-- Schema metadata, counters, timestamps
CREATE TABLE IF NOT EXISTS meta (
  key   TEXT PRIMARY KEY,
  value TEXT NOT NULL
);

-- Specs
CREATE TABLE IF NOT EXISTS specs (
  id           TEXT PRIMARY KEY,                -- "SPEC-42"
  slug         TEXT NOT NULL,
  title        TEXT NOT NULL,
  type         TEXT NOT NULL DEFAULT '{{DEFAULT_SPEC_TYPE}}' CHECK (type IN ({{SPEC_TYPES_CHECK}})),
  summary      TEXT NOT NULL,
  body         TEXT NOT NULL,
  path         TEXT NOT NULL,                   -- relative to workspace root
  position     INTEGER NOT NULL DEFAULT 0,      -- global order for specs list
  created_by   TEXT,
  updated_by   TEXT,
  content_hash TEXT NOT NULL,
  created_at   TEXT NOT NULL,
  updated_at   TEXT NOT NULL
);

-- Tasks
CREATE TABLE IF NOT EXISTS tasks (
  id           TEXT PRIMARY KEY,                -- "TASK-134"
  slug         TEXT NOT NULL,
  spec_id      TEXT NOT NULL REFERENCES specs(id) ON DELETE CASCADE,
  title        TEXT NOT NULL,
  status       TEXT NOT NULL CHECK (status IN ({{TASK_STAGES_CHECK}})),
  summary      TEXT NOT NULL,
  body         TEXT NOT NULL,
  path         TEXT NOT NULL,
  position     INTEGER NOT NULL DEFAULT 0,      -- per-status kanban order
  created_by   TEXT,
  updated_by   TEXT,
  content_hash TEXT NOT NULL,
  created_at   TEXT NOT NULL,
  updated_at   TEXT NOT NULL
);

-- Acceptance criteria (parsed from "## Acceptance criteria" section)
CREATE TABLE IF NOT EXISTS task_criteria (
  task_id    TEXT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
  position   INTEGER NOT NULL,                  -- 1-based order in the list
  text       TEXT NOT NULL,
  checked    INTEGER NOT NULL DEFAULT 0,
  PRIMARY KEY (task_id, position)
);

-- Undirected related links
CREATE TABLE IF NOT EXISTS spec_links (
  from_spec TEXT NOT NULL REFERENCES specs(id) ON DELETE CASCADE,
  to_spec   TEXT NOT NULL REFERENCES specs(id) ON DELETE CASCADE,
  PRIMARY KEY (from_spec, to_spec)
);

CREATE TABLE IF NOT EXISTS task_links (
  from_task TEXT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
  to_task   TEXT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
  PRIMARY KEY (from_task, to_task)
);

-- Directed task dependencies (blocker -> blocked)
CREATE TABLE IF NOT EXISTS task_dependencies (
  blocker_task TEXT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
  blocked_task TEXT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
  PRIMARY KEY (blocker_task, blocked_task)
);

-- KB documents
CREATE TABLE IF NOT EXISTS kb_docs (
  id           TEXT PRIMARY KEY,                -- "KB-17"
  slug         TEXT NOT NULL,
  title        TEXT NOT NULL,
  source_type  TEXT NOT NULL CHECK (source_type IN ('md','html','pdf','txt')),
  path         TEXT NOT NULL,                   -- original file path
  clean_path   TEXT,                            -- sanitized sidecar path for HTML
  note         TEXT,
  page_count   INTEGER,                         -- for PDFs only
  content_hash TEXT NOT NULL,
  added_at     TEXT NOT NULL,
  added_by     TEXT
);

-- KB chunks
CREATE TABLE IF NOT EXISTS kb_chunks (
  id         INTEGER PRIMARY KEY AUTOINCREMENT,
  doc_id     TEXT NOT NULL REFERENCES kb_docs(id) ON DELETE CASCADE,
  position   INTEGER NOT NULL,                  -- 0-based chunk index within doc
  text       TEXT NOT NULL,
  char_start INTEGER NOT NULL,
  char_end   INTEGER NOT NULL,
  page       INTEGER,                           -- nullable; PDF only
  UNIQUE (doc_id, position)
);

-- Citations (specs and tasks cite KB chunks)
CREATE TABLE IF NOT EXISTS citations (
  from_kind      TEXT NOT NULL CHECK (from_kind IN ('spec','task')),
  from_id        TEXT NOT NULL,
  kb_doc_id      TEXT NOT NULL REFERENCES kb_docs(id) ON DELETE CASCADE,
  chunk_position INTEGER NOT NULL,
  created_at     TEXT NOT NULL,
  PRIMARY KEY (from_kind, from_id, kb_doc_id, chunk_position)
);

-- Statistical chunk-to-chunk connections (TF-IDF cosine)
CREATE TABLE IF NOT EXISTS chunk_connections (
  from_chunk_id INTEGER NOT NULL REFERENCES kb_chunks(id) ON DELETE CASCADE,
  to_chunk_id   INTEGER NOT NULL REFERENCES kb_chunks(id) ON DELETE CASCADE,
  strength      REAL NOT NULL,                  -- cosine similarity 0..1
  method        TEXT NOT NULL DEFAULT 'tfidf_cosine',
  PRIMARY KEY (from_chunk_id, to_chunk_id)
);

-- Trash (soft delete with recovery)
CREATE TABLE IF NOT EXISTS trash (
  id            INTEGER PRIMARY KEY AUTOINCREMENT,
  kind          TEXT NOT NULL CHECK (kind IN ('spec','task','kb')),
  original_id   TEXT NOT NULL,
  original_path TEXT NOT NULL,
  content       BLOB NOT NULL,
  metadata      TEXT NOT NULL,                  -- JSON snapshot of primary rows
  deleted_at    TEXT NOT NULL,
  deleted_by    TEXT NOT NULL CHECK (deleted_by IN ('cli','watcher'))
);

-- Rejected files (manually created, not registered)
CREATE TABLE IF NOT EXISTS rejected_files (
  path        TEXT PRIMARY KEY,
  detected_at TEXT NOT NULL,
  reason      TEXT NOT NULL
);

-- FTS5 with BM25, porter stemming, prefix indexes
CREATE VIRTUAL TABLE IF NOT EXISTS specs_fts USING fts5(
  id UNINDEXED, title, type, summary, body,
  content='specs', content_rowid='rowid',
  tokenize='porter unicode61',
  prefix='2 3 4'
);

CREATE VIRTUAL TABLE IF NOT EXISTS tasks_fts USING fts5(
  id UNINDEXED, title, status UNINDEXED, summary, body,
  content='tasks', content_rowid='rowid',
  tokenize='porter unicode61',
  prefix='2 3 4'
);

CREATE VIRTUAL TABLE IF NOT EXISTS kb_chunks_fts USING fts5(
  text,
  content='kb_chunks', content_rowid='id',
  tokenize='porter unicode61',
  prefix='2 3 4'
);

-- Trigram fallback for fuzzy / substring matching
CREATE VIRTUAL TABLE IF NOT EXISTS search_trigram USING fts5(
  kind UNINDEXED,
  ref_id UNINDEXED,
  text,
  tokenize='trigram'
);

-- Triggers to keep FTS indexes in sync with base tables

-- specs_fts triggers
CREATE TRIGGER IF NOT EXISTS specs_ai AFTER INSERT ON specs BEGIN
  INSERT INTO specs_fts(rowid, id, title, type, summary, body)
    VALUES (NEW.rowid, NEW.id, NEW.title, NEW.type, NEW.summary, NEW.body);
  INSERT INTO search_trigram(kind, ref_id, text)
    VALUES ('spec', NEW.id, NEW.title || ' ' || NEW.summary || ' ' || NEW.body);
END;

CREATE TRIGGER IF NOT EXISTS specs_ad AFTER DELETE ON specs BEGIN
  INSERT INTO specs_fts(specs_fts, rowid, id, title, type, summary, body)
    VALUES ('delete', OLD.rowid, OLD.id, OLD.title, OLD.type, OLD.summary, OLD.body);
  DELETE FROM search_trigram WHERE kind = 'spec' AND ref_id = OLD.id;
END;

CREATE TRIGGER IF NOT EXISTS specs_au AFTER UPDATE ON specs BEGIN
  INSERT INTO specs_fts(specs_fts, rowid, id, title, type, summary, body)
    VALUES ('delete', OLD.rowid, OLD.id, OLD.title, OLD.type, OLD.summary, OLD.body);
  INSERT INTO specs_fts(rowid, id, title, type, summary, body)
    VALUES (NEW.rowid, NEW.id, NEW.title, NEW.type, NEW.summary, NEW.body);
  DELETE FROM search_trigram WHERE kind = 'spec' AND ref_id = OLD.id;
  INSERT INTO search_trigram(kind, ref_id, text)
    VALUES ('spec', NEW.id, NEW.title || ' ' || NEW.summary || ' ' || NEW.body);
END;

-- tasks_fts triggers
CREATE TRIGGER IF NOT EXISTS tasks_ai AFTER INSERT ON tasks BEGIN
  INSERT INTO tasks_fts(rowid, id, title, status, summary, body)
    VALUES (NEW.rowid, NEW.id, NEW.title, NEW.status, NEW.summary, NEW.body);
  INSERT INTO search_trigram(kind, ref_id, text)
    VALUES ('task', NEW.id, NEW.title || ' ' || NEW.summary || ' ' || NEW.body);
END;

CREATE TRIGGER IF NOT EXISTS tasks_ad AFTER DELETE ON tasks BEGIN
  INSERT INTO tasks_fts(tasks_fts, rowid, id, title, status, summary, body)
    VALUES ('delete', OLD.rowid, OLD.id, OLD.title, OLD.status, OLD.summary, OLD.body);
  DELETE FROM search_trigram WHERE kind = 'task' AND ref_id = OLD.id;
END;

CREATE TRIGGER IF NOT EXISTS tasks_au AFTER UPDATE ON tasks BEGIN
  INSERT INTO tasks_fts(tasks_fts, rowid, id, title, status, summary, body)
    VALUES ('delete', OLD.rowid, OLD.id, OLD.title, OLD.status, OLD.summary, OLD.body);
  INSERT INTO tasks_fts(rowid, id, title, status, summary, body)
    VALUES (NEW.rowid, NEW.id, NEW.title, NEW.status, NEW.summary, NEW.body);
  DELETE FROM search_trigram WHERE kind = 'task' AND ref_id = OLD.id;
  INSERT INTO search_trigram(kind, ref_id, text)
    VALUES ('task', NEW.id, NEW.title || ' ' || NEW.summary || ' ' || NEW.body);
END;

-- kb_chunks_fts triggers
CREATE TRIGGER IF NOT EXISTS kb_chunks_ai AFTER INSERT ON kb_chunks BEGIN
  INSERT INTO kb_chunks_fts(rowid, text)
    VALUES (NEW.id, NEW.text);
  INSERT INTO search_trigram(kind, ref_id, text)
    VALUES ('kb', NEW.doc_id, NEW.text);
END;

CREATE TRIGGER IF NOT EXISTS kb_chunks_ad AFTER DELETE ON kb_chunks BEGIN
  INSERT INTO kb_chunks_fts(kb_chunks_fts, rowid, text)
    VALUES ('delete', OLD.id, OLD.text);
  DELETE FROM search_trigram WHERE kind = 'kb' AND ref_id = OLD.doc_id AND text = OLD.text;
END;

CREATE TRIGGER IF NOT EXISTS kb_chunks_au AFTER UPDATE ON kb_chunks BEGIN
  INSERT INTO kb_chunks_fts(kb_chunks_fts, rowid, text)
    VALUES ('delete', OLD.id, OLD.text);
  INSERT INTO kb_chunks_fts(rowid, text)
    VALUES (NEW.id, NEW.text);
  DELETE FROM search_trigram WHERE kind = 'kb' AND ref_id = OLD.doc_id AND text = OLD.text;
  INSERT INTO search_trigram(kind, ref_id, text)
    VALUES ('kb', NEW.doc_id, NEW.text);
END;
