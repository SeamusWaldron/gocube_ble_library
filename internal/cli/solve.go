package cli

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/SeamusWaldron/gocube_ble_library/internal/recorder"
	"github.com/SeamusWaldron/gocube_ble_library/internal/storage"
)

var (
	solveNotes    string
	solveScramble string
	phaseKey      string
	phaseNotes    string
	listLimit     int
	showLast      bool
)

var solveCmd = &cobra.Command{
	Use:   "solve",
	Short: "Manage solve recording sessions",
	Long:  `Commands for starting, ending, and managing Rubik's Cube solve recordings.`,
}

var solveStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start a new solve recording",
	Long:  `Start a new solve recording session. The solve will begin recording moves from the GoCube.`,
	RunE:  runSolveStart,
}

var solveEndCmd = &cobra.Command{
	Use:   "end",
	Short: "End the current solve recording",
	Long:  `End the current solve recording session and compute derived statistics.`,
	RunE:  runSolveEnd,
}

var solvePhaseCmd = &cobra.Command{
	Use:   "phase",
	Short: "Mark a phase transition",
	Long: `Mark a phase transition during the current solve.

Available phases:
  1. inspection    - Optional inspection/orientation phase
  2. white_cross   - Building the white cross
  3. white_corners - Completing white corners
  4. middle_layer  - Completing the middle layer
  5. bottom_perm   - Positioning bottom layer pieces
  6. bottom_orient - Orienting bottom layer corners`,
	RunE: runSolvePhase,
}

var solveListCmd = &cobra.Command{
	Use:   "list",
	Short: "List recent solves",
	Long:  `Display a list of recent solve recordings with basic statistics.`,
	RunE:  runSolveList,
}

var solveShowCmd = &cobra.Command{
	Use:   "show [solve-id]",
	Short: "Show details of a solve",
	Long: `Display detailed information about a specific solve including:
- Solve metadata (duration, moves, TPS)
- Phase breakdown with timing
- Move sequence

Use --last to show the most recent solve.`,
	RunE: runSolveShow,
}

func init() {
	rootCmd.AddCommand(solveCmd)

	solveCmd.AddCommand(solveStartCmd)
	solveStartCmd.Flags().StringVar(&solveNotes, "notes", "", "Notes for this solve")
	solveStartCmd.Flags().StringVar(&solveScramble, "scramble", "", "Scramble sequence used")

	solveCmd.AddCommand(solveEndCmd)

	solveCmd.AddCommand(solvePhaseCmd)
	solvePhaseCmd.Flags().StringVar(&phaseKey, "phase", "", "Phase key (e.g., white_cross)")
	solvePhaseCmd.Flags().StringVar(&phaseNotes, "notes", "", "Notes for this phase mark")
	solvePhaseCmd.MarkFlagRequired("phase")

	solveCmd.AddCommand(solveListCmd)
	solveListCmd.Flags().IntVar(&listLimit, "limit", 20, "Maximum number of solves to display")

	solveCmd.AddCommand(solveShowCmd)
	solveShowCmd.Flags().BoolVar(&showLast, "last", false, "Show the most recent solve")
}

func runSolveStart(cmd *cobra.Command, args []string) error {
	// Open database
	db, err := openDB()
	if err != nil {
		return err
	}
	defer db.Close()

	// Load state
	stateFile, err := recorder.NewDefaultStateFile()
	if err != nil {
		return fmt.Errorf("failed to load state: %w", err)
	}

	// Check for existing active solve
	if stateFile.HasActiveSolve() {
		return fmt.Errorf("active solve already in progress: %s\nUse 'gocube solve end' to finish it first", stateFile.ActiveSolveID())
	}

	// Create session
	session := recorder.NewSession(db, stateFile)

	// Get device info if available
	state := stateFile.State()
	deviceName := state.LastDeviceName
	deviceID := state.LastDeviceID

	// Start solve
	solveID, err := session.Start(solveNotes, solveScramble, deviceName, deviceID, "0.1.0")
	if err != nil {
		return fmt.Errorf("failed to start solve: %w", err)
	}

	fmt.Printf("Started solve: %s\n", solveID)
	fmt.Println()
	fmt.Println("Phase marking:")
	fmt.Println("  gocube solve phase --phase white_cross")
	fmt.Println("  gocube solve phase --phase white_corners")
	fmt.Println("  gocube solve phase --phase middle_layer")
	fmt.Println("  gocube solve phase --phase bottom_perm")
	fmt.Println("  gocube solve phase --phase bottom_orient")
	fmt.Println()
	fmt.Println("Or use 'gocube solve record' for interactive mode")
	fmt.Println()
	fmt.Println("End with: gocube solve end")

	return nil
}

func runSolveEnd(cmd *cobra.Command, args []string) error {
	// Open database
	db, err := openDB()
	if err != nil {
		return err
	}
	defer db.Close()

	// Load state
	stateFile, err := recorder.NewDefaultStateFile()
	if err != nil {
		return fmt.Errorf("failed to load state: %w", err)
	}

	if !stateFile.HasActiveSolve() {
		return fmt.Errorf("no active solve in progress")
	}

	solveID := stateFile.ActiveSolveID()

	// Create session and resume
	session := recorder.NewSession(db, stateFile)
	if err := session.Resume(solveID); err != nil {
		return fmt.Errorf("failed to resume solve: %w", err)
	}

	// End solve
	if err := session.End(); err != nil {
		return fmt.Errorf("failed to end solve: %w", err)
	}

	// Get solve stats
	solveRepo := storage.NewSolveRepository(db)
	solve, err := solveRepo.Get(solveID)
	if err != nil {
		return fmt.Errorf("failed to get solve: %w", err)
	}

	moveCount, _ := solveRepo.GetMoveCount(solveID)

	fmt.Printf("Solve ended: %s\n", solveID)
	fmt.Println()
	if solve != nil && solve.DurationMs != nil {
		duration := time.Duration(*solve.DurationMs) * time.Millisecond
		fmt.Printf("Duration: %s\n", formatDuration(duration))
		fmt.Printf("Moves: %d\n", moveCount)
		if *solve.DurationMs > 0 {
			tps := float64(moveCount) / (float64(*solve.DurationMs) / 1000.0)
			fmt.Printf("TPS: %.2f\n", tps)
		}
	}
	fmt.Println()
	fmt.Printf("Generate report: gocube report solve --id %s\n", solveID)

	return nil
}

func runSolvePhase(cmd *cobra.Command, args []string) error {
	// Open database
	db, err := openDB()
	if err != nil {
		return err
	}
	defer db.Close()

	// Load state
	stateFile, err := recorder.NewDefaultStateFile()
	if err != nil {
		return fmt.Errorf("failed to load state: %w", err)
	}

	if !stateFile.HasActiveSolve() {
		return fmt.Errorf("no active solve in progress")
	}

	// Validate phase key
	phaseRepo := storage.NewPhaseRepository(db)
	phaseDef, err := phaseRepo.GetPhaseDef(phaseKey)
	if err != nil {
		return fmt.Errorf("invalid phase key '%s'\nUse one of: inspection, white_cross, white_corners, middle_layer, bottom_perm, bottom_orient", phaseKey)
	}

	// Create session and resume
	session := recorder.NewSession(db, stateFile)
	solveID := stateFile.ActiveSolveID()
	if err := session.Resume(solveID); err != nil {
		return fmt.Errorf("failed to resume solve: %w", err)
	}

	// Mark phase
	var notesPtr *string
	if phaseNotes != "" {
		notesPtr = &phaseNotes
	}

	if err := session.MarkPhase(phaseKey, notesPtr); err != nil {
		return fmt.Errorf("failed to mark phase: %w", err)
	}

	fmt.Printf("Marked phase: %s (%s)\n", phaseDef.DisplayName, phaseKey)
	fmt.Printf("Time: %s\n", formatDuration(time.Duration(session.ElapsedMs())*time.Millisecond))

	return nil
}

func runSolveList(cmd *cobra.Command, args []string) error {
	// Open database
	db, err := openDB()
	if err != nil {
		return err
	}
	defer db.Close()

	solveRepo := storage.NewSolveRepository(db)
	solves, err := solveRepo.List(listLimit)
	if err != nil {
		return fmt.Errorf("failed to list solves: %w", err)
	}

	if len(solves) == 0 {
		fmt.Println("No solves recorded yet")
		fmt.Println("Start a new solve with: gocube solve start")
		return nil
	}

	fmt.Printf("Recent solves (showing %d):\n", len(solves))
	fmt.Println()
	fmt.Printf("%-36s  %-20s  %-10s  %-6s  %-6s  %s\n", "ID", "Started", "Duration", "Moves", "TPS", "Notes")
	fmt.Println("------------------------------------  --------------------  ----------  ------  ------  -----")

	for _, s := range solves {
		duration := "-"
		moves := "-"
		tps := "-"

		if s.DurationMs != nil {
			d := time.Duration(*s.DurationMs) * time.Millisecond
			duration = formatDuration(d)
		}

		moveCount, _ := solveRepo.GetMoveCount(s.SolveID)
		if moveCount > 0 {
			moves = fmt.Sprintf("%d", moveCount)
			if s.DurationMs != nil && *s.DurationMs > 0 {
				tps = fmt.Sprintf("%.2f", float64(moveCount)/(float64(*s.DurationMs)/1000.0))
			}
		}

		notes := ""
		if s.Notes != nil {
			notes = *s.Notes
			if len(notes) > 30 {
				notes = notes[:27] + "..."
			}
		}

		status := ""
		if s.EndedAt == nil {
			status = " (active)"
		}

		fmt.Printf("%-36s  %-20s  %-10s  %-6s  %-6s  %s%s\n",
			s.SolveID,
			s.StartedAt.Format("2006-01-02 15:04:05"),
			duration,
			moves,
			tps,
			notes,
			status,
		)
	}

	return nil
}

func runSolveShow(cmd *cobra.Command, args []string) error {
	// Open database
	db, err := openDB()
	if err != nil {
		return err
	}
	defer db.Close()

	solveRepo := storage.NewSolveRepository(db)
	moveRepo := storage.NewMoveRepository(db)
	phaseRepo := storage.NewPhaseRepository(db)

	// Determine solve ID
	var solveID string
	if showLast {
		solves, err := solveRepo.List(1)
		if err != nil {
			return fmt.Errorf("failed to get latest solve: %w", err)
		}
		if len(solves) == 0 {
			return fmt.Errorf("no solves found")
		}
		solveID = solves[0].SolveID
	} else if len(args) > 0 {
		solveID = args[0]
	} else {
		return fmt.Errorf("please provide a solve ID or use --last")
	}

	// Get solve
	solve, err := solveRepo.Get(solveID)
	if err != nil {
		return fmt.Errorf("failed to get solve: %w", err)
	}
	if solve == nil {
		return fmt.Errorf("solve not found: %s", solveID)
	}

	// Get moves
	moves, err := moveRepo.GetBySolve(solveID)
	if err != nil {
		return fmt.Errorf("failed to get moves: %w", err)
	}

	// Get phase segments
	segments, err := phaseRepo.GetPhaseSegments(solveID)
	if err != nil {
		return fmt.Errorf("failed to get phases: %w", err)
	}

	// Display header
	fmt.Println("Solve Details")
	fmt.Println("=============")
	fmt.Println()

	// Basic info
	fmt.Printf("ID:      %s\n", solve.SolveID)
	fmt.Printf("Started: %s\n", solve.StartedAt.Format("2006-01-02 15:04:05"))
	if solve.EndedAt != nil {
		fmt.Printf("Ended:   %s\n", solve.EndedAt.Format("2006-01-02 15:04:05"))
	}
	if solve.Notes != nil && *solve.Notes != "" {
		fmt.Printf("Notes:   %s\n", *solve.Notes)
	}
	fmt.Println()

	// Calculate actual solve time (excluding scramble and inspection)
	var solveDurationMs int64
	var solveMoves int
	for _, seg := range segments {
		// Only count phases after inspection (actual solving)
		if seg.PhaseKey != "scramble" && seg.PhaseKey != "inspection" {
			solveDurationMs += seg.DurationMs
			solveMoves += seg.MoveCount
		}
	}

	// Stats
	fmt.Println("Statistics")
	fmt.Println("----------")
	if solveDurationMs > 0 {
		solveDuration := time.Duration(solveDurationMs) * time.Millisecond
		fmt.Printf("Solve Time: %s\n", formatDuration(solveDuration))
		if solveMoves > 0 {
			tps := float64(solveMoves) / (float64(solveDurationMs) / 1000.0)
			fmt.Printf("TPS:        %.2f\n", tps)
		}
	}
	fmt.Printf("Moves:      %d\n", solveMoves)
	if solve.DurationMs != nil {
		sessionDuration := time.Duration(*solve.DurationMs) * time.Millisecond
		fmt.Printf("Session:    %s (includes scramble/inspection)\n", formatDuration(sessionDuration))
	}
	fmt.Println()

	// Phase breakdown with moves
	if len(segments) > 0 {
		fmt.Println("Phases")
		fmt.Println("------")

		for _, seg := range segments {
			duration := formatDuration(time.Duration(seg.DurationMs) * time.Millisecond)
			tps := ""
			if seg.TPS > 0 {
				tps = fmt.Sprintf(" @ %.2f TPS", seg.TPS)
			}

			// Phase header
			fmt.Printf("\n%s (%d moves, %s%s)\n",
				storage.PhaseDisplayName(seg.PhaseKey),
				seg.MoveCount,
				duration,
				tps,
			)

			// Get moves for this phase
			phaseMoves, _ := moveRepo.GetBySolveRange(solveID, seg.StartTsMs, seg.EndTsMs)
			if len(phaseMoves) > 0 {
				// Group moves into lines of ~60 chars
				var line string
				for i, m := range phaseMoves {
					if len(line)+len(m.Notation)+1 > 60 {
						fmt.Printf("  %s\n", line)
						line = m.Notation
					} else if line == "" {
						line = m.Notation
					} else {
						line += " " + m.Notation
					}

					// Print last line
					if i == len(phaseMoves)-1 && line != "" {
						fmt.Printf("  %s\n", line)
					}
				}
			}
		}
	} else if len(moves) > 0 {
		// No phases, just show all moves
		fmt.Println("Moves")
		fmt.Println("-----")

		var line string
		for i, m := range moves {
			if len(line)+len(m.Notation)+1 > 60 {
				fmt.Println(line)
				line = m.Notation
			} else if line == "" {
				line = m.Notation
			} else {
				line += " " + m.Notation
			}

			if i == len(moves)-1 && line != "" {
				fmt.Println(line)
			}
		}
	}

	return nil
}

func openDB() (*storage.DB, error) {
	path := getDBPath()
	var db *storage.DB
	var err error

	if path == "" {
		db, err = storage.OpenDefault()
	} else {
		db, err = storage.Open(path)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.MigrateUp(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	return db, nil
}

func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%.2fs", d.Seconds())
	}
	mins := int(d.Minutes())
	secs := d.Seconds() - float64(mins*60)
	return fmt.Sprintf("%dm%.1fs", mins, secs)
}
