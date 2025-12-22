-- GoCube Solve Recorder Schema v4
-- Migration: 004_orientations
-- Adds dedicated orientation tracking table

-- Orientation state changes during a solve.
-- Records when the discrete orientation (up_face/front_face) changes.
CREATE TABLE IF NOT EXISTS orientations (
  orientation_id  INTEGER PRIMARY KEY AUTOINCREMENT,
  solve_id        TEXT NOT NULL,
  ts_ms           INTEGER NOT NULL,           -- ms since solve start
  up_face         TEXT NOT NULL CHECK (up_face IN ('U','D','F','B','R','L')),
  front_face      TEXT NOT NULL CHECK (front_face IN ('U','D','F','B','R','L')),
  source_event_id INTEGER,                    -- optional linkage to events.event_id
  FOREIGN KEY (solve_id) REFERENCES solves(solve_id) ON DELETE CASCADE,
  FOREIGN KEY (source_event_id) REFERENCES events(event_id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_orientations_solve_ts
  ON orientations(solve_id, ts_ms);

-- Record migration version
INSERT OR REPLACE INTO schema_version(version, applied_at)
VALUES (4, datetime('now'));
