-- GoCube Solve Recorder Schema v2
-- Migration: 002_new_phases
-- Updates phase definitions to match 80s layer-by-layer method

-- Add 'algo_start' as valid mark_type for algorithm markers
-- SQLite doesn't support ALTER CHECK, so we'll handle this in code

-- Clear old phase definitions and add new ones
DELETE FROM phase_defs;

INSERT INTO phase_defs(phase_key, display_name, order_index, description)
VALUES
  ('inspection',       'Inspection',           0,  'Pre-move inspection/orientation'),
  ('white_cross',      'White Cross',          1,  'Build the white cross on top'),
  ('top_corners',      'Top Corners',          2,  'Complete the top layer corners'),
  ('middle_layer',     'Middle Layer',         3,  'Complete the middle ring'),
  ('middle_rhs',       'Middle RHS Algo',      31, 'Right-hand side middle piece algorithm'),
  ('middle_lhs',       'Middle LHS Algo',      32, 'Left-hand side middle piece algorithm'),
  ('bottom_cross',     'Bottom Cross',         4,  'Create the yellow cross on bottom'),
  ('position_corners', 'Position Corners',     5,  'Move corners to correct positions'),
  ('rotate_corners',   'Rotate Corners',       6,  'Rotate corners to correct orientation'),
  ('complete',         'Complete',             7,  'Solve complete');

-- Record migration version
INSERT OR REPLACE INTO schema_version(version, applied_at)
VALUES (2, datetime('now'));
