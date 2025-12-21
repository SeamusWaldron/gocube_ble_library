-- GoCube Solve Recorder Schema v3
-- Migration: 003_add_scramble
-- Adds scramble phase for recording scramble moves before inspection

INSERT OR IGNORE INTO phase_defs(phase_key, display_name, order_index, description)
VALUES
  ('scramble', 'Scramble', -1, 'Scrambling the cube before solve');

-- Record migration version
INSERT OR REPLACE INTO schema_version(version, applied_at)
VALUES (3, datetime('now'));
