-- GoCube Solve Recorder Schema v1
-- Migration: 001_initial

PRAGMA foreign_keys = ON;

-- =========================
-- Core solve/session tables
-- =========================

CREATE TABLE IF NOT EXISTS solves (
  solve_id        TEXT PRIMARY KEY,               -- UUID
  started_at      TEXT NOT NULL,                  -- ISO8601 UTC
  ended_at        TEXT,                           -- ISO8601 UTC
  duration_ms     INTEGER,                        -- computed at end
  scramble_text   TEXT,                           -- optional manual entry
  notes           TEXT,
  device_name     TEXT,
  device_id       TEXT,
  app_version     TEXT
);

CREATE INDEX IF NOT EXISTS idx_solves_started_at
  ON solves(started_at);

-- Raw events from BLE notifications, plus lifecycle events (connect/disconnect).
CREATE TABLE IF NOT EXISTS events (
  event_id            INTEGER PRIMARY KEY AUTOINCREMENT,
  solve_id            TEXT NOT NULL,
  ts_ms               INTEGER NOT NULL,           -- ms since solve start
  event_type          TEXT NOT NULL,              -- rotation/orientation/battery/etc
  payload_json        TEXT NOT NULL,              -- decoded fields (keep everything)
  raw_payload_base64  TEXT,                       -- optional exact raw bytes
  FOREIGN KEY (solve_id) REFERENCES solves(solve_id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_events_solve_ts
  ON events(solve_id, ts_ms);

CREATE INDEX IF NOT EXISTS idx_events_type
  ON events(event_type);

-- Derived move stream for analysis.
-- face: one of R,L,U,D,F,B (canonical cube notation)
-- turn:  1  => clockwise quarter
--       -1  => anticlockwise quarter
--        2  => 180
CREATE TABLE IF NOT EXISTS moves (
  move_id          INTEGER PRIMARY KEY AUTOINCREMENT,
  solve_id         TEXT NOT NULL,
  move_index       INTEGER NOT NULL,
  ts_ms            INTEGER NOT NULL,              -- ms since solve start
  face             TEXT NOT NULL CHECK (face IN ('R','L','U','D','F','B')),
  turn             INTEGER NOT NULL CHECK (turn IN (-1, 1, 2)),
  notation         TEXT NOT NULL,                 -- e.g. R, R', R2
  source_event_id  INTEGER,                       -- optional linkage to events.event_id
  FOREIGN KEY (solve_id) REFERENCES solves(solve_id) ON DELETE CASCADE,
  FOREIGN KEY (source_event_id) REFERENCES events(event_id) ON DELETE SET NULL,
  UNIQUE (solve_id, move_index)
);

CREATE INDEX IF NOT EXISTS idx_moves_solve_idx
  ON moves(solve_id, move_index);

CREATE INDEX IF NOT EXISTS idx_moves_solve_ts
  ON moves(solve_id, ts_ms);

-- =========================
-- Phase configuration + marks
-- =========================

CREATE TABLE IF NOT EXISTS phase_defs (
  phase_key     TEXT PRIMARY KEY,                 -- e.g. white_cross
  display_name  TEXT NOT NULL,                    -- e.g. White cross
  order_index   INTEGER NOT NULL,                 -- display order
  description   TEXT,
  is_active     INTEGER NOT NULL DEFAULT 1
);

-- Manual phase marks.
-- Recommended model: mark_type='start' meaning "phase X begins now"
CREATE TABLE IF NOT EXISTS phase_marks (
  phase_mark_id  INTEGER PRIMARY KEY AUTOINCREMENT,
  solve_id       TEXT NOT NULL,
  ts_ms          INTEGER NOT NULL,
  phase_key      TEXT NOT NULL,
  mark_type      TEXT NOT NULL CHECK (mark_type IN ('start')),
  notes          TEXT,
  FOREIGN KEY (solve_id) REFERENCES solves(solve_id) ON DELETE CASCADE,
  FOREIGN KEY (phase_key) REFERENCES phase_defs(phase_key)
);

CREATE INDEX IF NOT EXISTS idx_phase_marks_solve_ts
  ON phase_marks(solve_id, ts_ms);

-- Optional cached segments derived from phase marks.
CREATE TABLE IF NOT EXISTS derived_phase_segments (
  segment_id     INTEGER PRIMARY KEY AUTOINCREMENT,
  solve_id       TEXT NOT NULL,
  phase_key      TEXT NOT NULL,
  start_ts_ms    INTEGER NOT NULL,
  end_ts_ms      INTEGER NOT NULL,
  duration_ms    INTEGER NOT NULL,
  move_count     INTEGER NOT NULL,
  tps            REAL NOT NULL,                   -- turns per second in the segment
  FOREIGN KEY (solve_id) REFERENCES solves(solve_id) ON DELETE CASCADE,
  FOREIGN KEY (phase_key) REFERENCES phase_defs(phase_key)
);

CREATE INDEX IF NOT EXISTS idx_segments_solve_phase
  ON derived_phase_segments(solve_id, phase_key);

-- Optional analysis cache for expensive derived reports (v2).
CREATE TABLE IF NOT EXISTS analysis_cache (
  cache_id     INTEGER PRIMARY KEY AUTOINCREMENT,
  solve_id     TEXT NOT NULL,
  cache_key    TEXT NOT NULL,                     -- e.g. ngram_4_14
  created_at   TEXT NOT NULL,                     -- ISO8601 UTC
  payload_json TEXT NOT NULL,
  FOREIGN KEY (solve_id) REFERENCES solves(solve_id) ON DELETE CASCADE,
  UNIQUE (solve_id, cache_key)
);

-- Schema version tracking
CREATE TABLE IF NOT EXISTS schema_version (
  version     INTEGER PRIMARY KEY,
  applied_at  TEXT NOT NULL                       -- ISO8601 UTC
);

-- =========================
-- Seed phase definitions
-- =========================

INSERT OR IGNORE INTO phase_defs(phase_key, display_name, order_index, description)
VALUES
  ('inspection',     'Inspection / orientation', 10, 'Optional: pre-move inspection/orientation'),
  ('white_cross',    'White cross',              20, 'Build the white cross on top'),
  ('white_corners',  'White corners',            30, 'Complete the top layer (white face)'),
  ('middle_layer',   'Middle layer',             40, 'Complete the middle ring'),
  ('bottom_perm',    'Bottom permutation',       50, 'All pieces positioned correctly'),
  ('bottom_orient',  'Bottom corner orientation',60, 'Final: orient remaining corners');

-- Record migration version
INSERT OR IGNORE INTO schema_version(version, applied_at)
VALUES (1, datetime('now'));
