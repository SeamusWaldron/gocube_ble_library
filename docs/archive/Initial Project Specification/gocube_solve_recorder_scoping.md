# GoCube Solve Recorder and 80s Method Optimiser (macOS, CLI-first)

This document scopes a CLI tool that records GoCube solves (manual start/end, manual phase marking) and generates analysis outputs to help formalise and optimise an 80s top-down Rubik’s Cube method.

## Goals

- Record complete solve sessions from a GoCube smart cube on macOS.
- Allow **manual Start** and **manual End** of a solve (no automatic solved detection required for v1).
- Allow **manual phase marking** during the solve (your top-down phases).
- Persist raw BLE events and derived move streams in **SQLite**.
- Produce actionable analysis outputs:
  - per-solve metrics and phase metrics
  - repetition detection
  - n-gram mining (surface repeated “micro-algorithms”)
  - final-phase tool detection (forward/reverse + mirror)
  - trends across solves

## Non-goals (v1)

- Automatic solved detection
- Automatic phase inference (manual marking first)
- 3D cube rendering
- Mobile app / Unity dependency
- Cloud sync (keep it local and fast)

## CLI user workflow (v1)

1. Connect to the GoCube over BLE.
2. Start a new solve **manually**.
3. Mark phases **manually** as you progress.
4. End the solve **manually**.
5. Generate reports for the last solve or any solve ID.

### Example CLI usage

```bash
# Pair/connect and show status
gocube status

# Start a new solve session (returns solve_id)
gocube solve start --notes "baseline 80s method"

# Mark phases during solve (hotkeys can be added later; v1 uses commands)
gocube solve phase --phase white_cross
gocube solve phase --phase white_corners
gocube solve phase --phase middle_layer
gocube solve phase --phase bottom_perm
gocube solve phase --phase bottom_orient

# End the solve
gocube solve end

# List recent solves
gocube solve list --limit 20

# Generate reports
gocube report solve --last
gocube report solve --id <solve_id>
gocube report trend --window 50

# Export moves
gocube export moves --id <solve_id> --format txt
gocube export moves --id <solve_id> --format json
```

Notes:
- For ergonomics, you can later add a “listen for keypresses” mode (e.g. `gocube solve record`) to accept `1..6` for phases. The data model below already supports it.

## Phases (default, editable)

You can edit phase names later without breaking data by changing `phase_defs`.

1. `inspection` (optional)
2. `white_cross`
3. `white_corners` (complete top layer)
4. `middle_layer`
5. `bottom_perm` (all pieces positioned correctly)
6. `bottom_orient` (final corner orientation phase)

## Architecture (simple and robust)

- **Recorder (Go)**
  - BLE connection + decoding using GoCube protocol
  - Emits raw events (rotation notifications, etc.)
  - Derives canonical moves (face + quarter-turn direction)
- **Storage (SQLite)**
  - Append-only-ish event capture
  - Derived move table for analysis
  - Manual phase marks
- **Analysis (Go)**
  - Runs on demand (CLI `report` commands)
  - Optionally caches derived phase segments

## Data model (SQLite)

Design principles:
- Store **raw events** for reproducibility.
- Store **canonical move stream** for analysis.
- Store **manual annotations** (phase marks).
- Keep schema migration-friendly.

### Tables

- `solves` — one row per solve session
- `events` — raw BLE notifications and other telemetry
- `moves` — derived canonical move stream
- `phase_defs` — phase definitions (editable)
- `phase_marks` — manual phase markers during a solve
- `derived_phase_segments` — optional cached segments per solve/phase
- `analysis_cache` — optional blobs for expensive reports (v2)

---

# Concrete SQLite schema.sql (v1)

Save the following as `schema.sql` and execute once to create the database.

```sql
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
  solve_id             TEXT NOT NULL,
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
```

---

# Canonical moves vs your notation

Store canonical moves internally (`R,L,U,D,F,B` + direction). Export layers can render either:

- Standard cube notation (`R' D2 ...`)
- Your physical notation (“R down”, “B rotate right x 2”, etc.)

Keeping analysis canonical prevents future confusion if you change wording.

---

# Analysis outputs (exact artefacts)

All reports are generated by CLI commands and written to a directory (default: `./reports/<solve_id>/`).

## A) Per-solve summary
File: `solve_summary.json`

Fields:
- `solve_id`
- `started_at`, `ended_at`, `duration_ms`
- `total_moves`
- `tps_overall` = `total_moves / (duration_ms/1000)`
- `phase_stats[]` (in order):
  - `phase_key`
  - `start_ts_ms`, `end_ts_ms`, `duration_ms`
  - `move_count`
  - `tps`
- `longest_pause_ms` (max gap between moves)
- `pause_count_over_threshold` (e.g. >1500ms)
- `notes`

## B) Move stream export
Files:
- `moves.txt` — single-line notation: `R' D2 R ...`
- `moves.json` — array: `{move_index, ts_ms, face, turn, notation}`

## C) Phase move lists
Directory: `phase_moves/`
Files: `phase_moves/<phase_key>.txt`

Each file contains the move sequence for that phase only.

## D) Repetition and wasted motion
File: `repetition_report.json`

Contains:
- `immediate_cancellations[]` (examples with indices)
  - pattern: move followed immediately by its inverse (e.g. `R` then `R'`)
- `merge_opportunities[]`
  - adjacent same-face moves that should merge (`D` + `D` -> `D2`)
- `back_and_forth_patterns[]`
  - high-frequency alternating patterns (useful for “search” detection)
- `top_k_ngrams[]`
  - mined from lengths 4..14
  - each entry includes:
    - `n`
    - `sequence` (notation list)
    - `count`
    - `occurrences` (first few start indices + timestamps)

## E) Final-phase tool detection
File: `final_phase_report.json`

Based on manual `bottom_orient` phase boundaries:
- `final_phase_move_count`
- `final_phase_duration_ms`
- `rhs_forward_count`
- `rhs_reverse_count`
- `lhs_forward_count`
- `lhs_reverse_count`
- `consecutive_tool_repeats` (e.g. forward followed by forward)
- `time_between_tool_runs_ms[]` (inspection/decision latency proxy)

Matching method (v1): exact match against canonical sequences for each tool.

## F) Trends across solves
File: `trend_report.json`

For a chosen window (e.g. last 50):
- `window_size`
- `solves[]` (id, timestamp, duration_ms, total_moves)
- rolling averages:
  - total moves, duration, TPS
  - moves per phase
  - time per phase
- “most repeated n-gram” overall in window
- “most improved phase” vs earliest in window

---

# N-gram mining approach (efficient for thousands of solves)

Goal: find repeated move sequences (length 4..14) across a solve and across many solves, without excessive memory or runtime.

## Representation

Convert each move to a small integer token:
- face in {R,L,U,D,F,B} -> 0..5
- turn in {-1,1,2} -> map to 0..2
- token = face*3 + turn_code  -> 0..17

Store per solve as `[]uint8` tokens (fast).

## Per-solve n-gram mining (fast, low memory)

For each solve:
1. Read tokens from `moves` ordered by `move_index`.
2. For each n in [4..14]:
   - Use a rolling hash to count n-grams in O(moves) time.
   - Keep only the top K by count (e.g. K=50 per n) to limit output.
3. Output top n-grams with example occurrences.

### Rolling hash detail (Rabin–Karp)

Let tokens be `t[0..M-1]`.

For a fixed n:
- use a 64-bit rolling hash with overflow arithmetic for speed
- keep a power term `B^(n-1)` precomputed per n

Update step for window starting at i:
- `h_next = (h - t[i]*pow) * B + t[i+n]` (all in uint64 overflow)

Collision handling:
- store the first occurrence’s token slice (n bytes) as the representative
- when a hash hits an existing entry, compare slices; if mismatch, treat as a separate entry (rare)

Data structure per n:
- `map[uint64]Entry` where Entry includes:
  - `count`
  - `repr []uint8` (length n)
  - `sample_indices []int` (cap to 10)
  - `first_ts_ms` (optional)

Complexity per solve:
- O(M * 11) updates, where M is number of moves
- With M ~ 100–300 typical, this is tiny
- Works comfortably for thousands of solves

## Cross-solve aggregation (recommended approach)

Rather than counting every n-gram for every solve globally (heavy), do:

1. Per solve: compute top K n-grams per n.
2. Merge into global maps keyed by `(n, sequence_bytes)`:
   - `global_count += count_in_solve`
   - `solves_seen += 1`
3. Keep only global top N per n (e.g. N=100) for output.

This gives you:
- “which micro-algorithms dominate my method overall?”
- without the cost of exhaustive mining.

## Output shaping

For each top n-gram:
- render to standard notation list
- include count and a few example occurrences (solve_id + start index)
- optional: render to your physical notation later

---

# CLI command set (v1)

## `gocube status`
Shows BLE connection state, cube identity, battery if available.

## `gocube solve start [--notes ...] [--scramble ...]`
Creates a `solves` row and sets an active solve id. Begins recording.

## `gocube solve phase --phase <phase_key> [--notes ...]`
Writes a phase mark event at current elapsed time.

## `gocube solve end`
Finalises solve, computes duration, optionally computes derived phase segments, clears active solve id.

## `gocube solve list [--limit N]`
Lists recent solves.

## `gocube report solve (--last | --id <solve_id>)`
Generates per-solve reports.

## `gocube report trend --window N`
Generates trend report.

## `gocube export moves --id <solve_id> --format (txt|json)`
Exports moves.

---

# Minimal iteration plan

## Milestone 1
- Reliable move capture
- Manual start/end
- Phase marking
- `solve_summary.json` and `moves.txt`

## Milestone 2
- Repetition report
- Phase move lists

## Milestone 3
- N-gram mining
- Final-phase tool detection
- Trend report

---

# Implementation notes

- Keep an app state file, e.g. `~/.gocube_recorder/state.json`:
  - `db_path`
  - `active_solve_id`
  - last connected device id
- Recorder resilience:
  - reconnect loop
  - record disconnect events
- Keep event decoding versioned inside `payload_json` so old data stays usable.
