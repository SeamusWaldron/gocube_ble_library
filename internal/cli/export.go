package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/seamusw/gocube/internal/storage"
)

var (
	exportSolveID string
	exportFormat  string
	exportOutput  string
	exportLast    bool
)

var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export solve data",
	Long:  `Export solve data in various formats.`,
}

var exportMovesCmd = &cobra.Command{
	Use:   "moves",
	Short: "Export moves from a solve",
	Long: `Export the move sequence from a solve in text or JSON format.

Examples:
  gocube export moves --last
  gocube export moves --id <solve_id> --format json
  gocube export moves --id <solve_id> --format txt -o moves.txt`,
	RunE: runExportMoves,
}

func init() {
	rootCmd.AddCommand(exportCmd)

	exportCmd.AddCommand(exportMovesCmd)
	exportMovesCmd.Flags().StringVar(&exportSolveID, "id", "", "Solve ID to export")
	exportMovesCmd.Flags().BoolVar(&exportLast, "last", false, "Export the last solve")
	exportMovesCmd.Flags().StringVar(&exportFormat, "format", "txt", "Export format (txt, json)")
	exportMovesCmd.Flags().StringVarP(&exportOutput, "output", "o", "", "Output file (default: stdout)")
}

func runExportMoves(cmd *cobra.Command, args []string) error {
	if exportSolveID == "" && !exportLast {
		return fmt.Errorf("specify --id or --last")
	}

	// Open database
	db, err := openDB()
	if err != nil {
		return err
	}
	defer db.Close()

	// Get solve ID
	solveID := exportSolveID
	if exportLast {
		solveRepo := storage.NewSolveRepository(db)
		solve, err := solveRepo.GetLast()
		if err != nil {
			return fmt.Errorf("failed to get last solve: %w", err)
		}
		if solve == nil {
			return fmt.Errorf("no solves found")
		}
		solveID = solve.SolveID
	}

	// Get moves
	moveRepo := storage.NewMoveRepository(db)
	moves, err := moveRepo.GetBySolve(solveID)
	if err != nil {
		return fmt.Errorf("failed to get moves: %w", err)
	}

	if len(moves) == 0 {
		return fmt.Errorf("no moves found for solve %s", solveID)
	}

	// Format output
	var output string

	switch strings.ToLower(exportFormat) {
	case "txt":
		var notations []string
		for _, m := range moves {
			notations = append(notations, m.Notation)
		}
		output = strings.Join(notations, " ")

	case "json":
		type MoveJSON struct {
			MoveIndex int    `json:"move_index"`
			TsMs      int64  `json:"ts_ms"`
			Face      string `json:"face"`
			Turn      int    `json:"turn"`
			Notation  string `json:"notation"`
		}

		var movesJSON []MoveJSON
		for _, m := range moves {
			movesJSON = append(movesJSON, MoveJSON{
				MoveIndex: m.MoveIndex,
				TsMs:      m.TsMs,
				Face:      m.Face,
				Turn:      m.Turn,
				Notation:  m.Notation,
			})
		}

		data, err := json.MarshalIndent(movesJSON, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		output = string(data)

	default:
		return fmt.Errorf("unknown format: %s (use txt or json)", exportFormat)
	}

	// Write output
	if exportOutput == "" {
		fmt.Println(output)
	} else {
		// Ensure directory exists
		dir := filepath.Dir(exportOutput)
		if dir != "" && dir != "." {
			if err := os.MkdirAll(dir, 0755); err != nil {
				return fmt.Errorf("failed to create output directory: %w", err)
			}
		}

		if err := os.WriteFile(exportOutput, []byte(output+"\n"), 0644); err != nil {
			return fmt.Errorf("failed to write output file: %w", err)
		}

		fmt.Printf("Exported %d moves to %s\n", len(moves), exportOutput)
	}

	return nil
}
