ALTER TABLE tags ADD COLUMN downloads INTEGER NOT NULL DEFAULT 0;
ALTER TABLE tags ADD COLUMN updated_at TEXT NOT NULL DEFAULT '';

UPDATE tags SET updated_at = datetime('now') WHERE updated_at = '';
