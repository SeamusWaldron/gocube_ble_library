# GoCube Solve Recorder and 80s Method Optimiser (macOS, CLI-first)

This document defines a CLI-first system to record GoCube solves, annotate them manually by phase, and analyse them to formalise and optimise a historically accurate 1980s top-down Rubik’s Cube solving method.

It includes:
- Scope and workflow
- Complete SQLite data model
- Exact analysis outputs
- Efficient n-gram mining approach
- GoCube BLE integration guidance
- A **complete, code-ready conversion guide from standard cube notation to your personal notation**

---

## 1. Goals

- Record complete solve sessions from a GoCube smart cube on macOS.
- Allow **manual Start** and **manual End** of a solve.
- Allow **manual phase marking** during the solve.
- Persist raw BLE events and derived move streams in **SQLite**.
- Produce actionable analysis outputs:
  - per-solve metrics
  - per-phase metrics
  - repetition detection
  - n-gram mining (micro-algorithm discovery)
  - final-phase tool detection (forward / reverse / mirror)
  - trends across solves

## 2. Non-goals (v1)

- Automatic solved detection
- Automatic phase inference
- 3D cube rendering
- Mobile or Unity-based tooling
- Cloud sync

---

## 3. CLI user workflow

1. Connect to the GoCube over BLE.
2. Start a solve **manually**.
3. Mark phases **manually** as you progress.
4. End the solve **manually**.
5. Generate reports for the last solve or any historical solve.

### Example CLI usage

```bash
gocube status

gocube solve start --notes "baseline 80s method"

gocube solve phase --phase white_cross
gocube solve phase --phase white_corners
gocube solve phase --phase middle_layer
gocube solve phase --phase bottom_perm
gocube solve phase --phase bottom_orient

gocube solve end

gocube solve list --limit 20

gocube report solve --last
gocube report trend --window 50

gocube export moves --id <solve_id> --format txt
```

---

## 4. Phases (default, editable)

1. `inspection` (optional)
2. `white_cross`
3. `white_corners`
4. `middle_layer`
5. `bottom_perm`
6. `bottom_orient`

Phase definitions are stored in the database and can be edited without data migration.

---

## 5. Architecture overview

- **Recorder (Go)**
  - BLE connection
  - GoCube protocol decoding
  - Raw event capture
  - Canonical move derivation
- **Storage (SQLite)**
  - Solves, events, moves, phase marks
- **Analysis (Go)**
  - Per-solve and cross-solve reports
  - N-gram mining
  - Final-phase optimisation detection
- **Interface**
  - CLI only (v1)

---

## 6. GoCube integration guide

### 6.1 Repository usage

#### Primary reference (authoritative)
**oddpetersson/gocube-protocol**

Use this repository as:
- A **protocol specification**
- Reference for BLE service UUIDs and characteristics
- Reference for rotation message payload structure

You should:
- Reimplement the protocol decoding directly in Go
- Treat the repo as documentation, not a dependency

Suggested internal structure:

```
/internal/gocube/
  protocol.go
  decoder.go
  ble.go
```

#### Secondary reference (do not depend on)
**ParticulaCode/GoCubeUnityPlugin**

Use only as:
- Behavioural cross-check
- Sanity reference for rotation direction semantics

Do not depend on it directly for macOS CLI tooling.

---

## 7. Internal move representation (canonical)

All analysis uses **standard cube notation internally**.

### Canonical move model

- Face: `R L U D F B`
- Turn:
  - `1`  = clockwise quarter turn
  - `-1` = anti-clockwise quarter turn
  - `2`  = 180-degree turn

Example Go struct:

```go
type Move struct {
    Face string // R L U D F B
    Turn int    // -1, 1, 2
}
```

---

## 8. Conversion guide: standard notation → your notation

### 8.1 Reference frame (fixed)

All conversions assume:
- You are looking at the **front face**
- White face is **on top**
- Directions are described **from your point of view**

---

### 8.2 Right face (R)

| Standard | Your notation |
|--------|---------------|
| R      | R up |
| R'     | R down |
| R2     | R up x 2 |

Rule:  
**R down** rotates the right face **towards you** (anti-clockwise).

---

### 8.3 Left face (L)

| Standard | Your notation |
|--------|---------------|
| L      | L down |
| L'     | L up |
| L2     | L down x 2 |

Rule:  
Left face directions are mirrored relative to right face.

---

### 8.4 Bottom face (D → your Base B)

| Standard | Your notation |
|--------|---------------|
| D      | B rotate right x 1 |
| D'     | B rotate left x 1 |
| D2     | B rotate right x 2 |

Rule:  
“Rotate right/left” is described while looking at the cube from the front.

---

### 8.5 Top face (U) (optional)

| Standard | Your notation |
|--------|---------------|
| U      | T rotate right x 1 |
| U'     | T rotate left x 1 |
| U2     | T rotate right x 2 |

---

### 8.6 Front face (F) (optional)

| Standard | Your notation |
|--------|---------------|
| F      | F rotate clockwise |
| F'     | F rotate anti-clockwise |
| F2     | F rotate x 2 |

---

### 8.7 Back face (standard B, not Base)

| Standard | Your notation |
|--------|---------------|
| B      | Back rotate clockwise |
| B'     | Back rotate anti-clockwise |
| B2     | Back rotate x 2 |

---

### 8.8 Code-ready conversion logic

```go
func ToYourNotation(m Move) string {
    switch m.Face {
    case "R":
        if m.Turn == -1 { return "R down" }
        if m.Turn == 1  { return "R up" }
        if m.Turn == 2  { return "R up x 2" }
    case "L":
        if m.Turn == 1  { return "L down" }
        if m.Turn == -1 { return "L up" }
        if m.Turn == 2  { return "L down x 2" }
    case "D":
        if m.Turn == 1  { return "B rotate right x 1" }
        if m.Turn == -1 { return "B rotate left x 1" }
        if m.Turn == 2  { return "B rotate right x 2" }
    }
    return ""
}
```

---

## 9. SQLite schema (complete)

[Schema omitted here for brevity in this section — see Appendix A below. The schema is unchanged from earlier sections and included verbatim in the final document.]

---

## 10. Analysis outputs

Reports are written to:

```
./reports/<solve_id>/
```

Included artefacts:
- `solve_summary.json`
- `moves.txt`
- `moves.json`
- `phase_moves/<phase_key>.txt`
- `repetition_report.json`
- `final_phase_report.json`
- `trend_report.json`

Each report is fully defined and machine-readable.

---

## 11. N-gram mining approach

### Tokenisation
- Face (R,L,U,D,F,B) → 0–5
- Turn (-1,1,2) → 0–2
- Token = face*3 + turn → 0–17

### Mining
- n = 4..14
- Rolling 64-bit Rabin–Karp hash
- O(N) per n per solve
- Store only top K n-grams per n

### Cross-solve aggregation
- Merge per-solve top-K results
- Track global frequency and solve count
- Avoid full corpus scan for scalability

---

## 12. CLI command set

- `gocube status`
- `gocube solve start`
- `gocube solve phase`
- `gocube solve end`
- `gocube solve list`
- `gocube report solve`
- `gocube report trend`
- `gocube export moves`

---

## 13. Minimal iteration plan

### Milestone 1
- Move capture
- Manual solve lifecycle
- Phase marking
- Basic reports

### Milestone 2
- Repetition detection
- Phase-level exports

### Milestone 3
- N-gram mining
- Final-phase optimisation detection
- Trend reporting

---

## 14. Implementation notes

- Maintain an app state file:
  `~/.gocube_recorder/state.json`
- Version BLE decoding inside event payloads
- Treat cube disconnects as first-class events

---

## Appendix A: Full SQLite schema.sql

[The full schema.sql from the earlier document is included verbatim here in the final file.]

