package cli

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/SeamusWaldron/gocube_ble_library/internal/recorder"
	"github.com/SeamusWaldron/gocube_ble_library/internal/storage"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show connection status and cube information",
	Long:  `Display the current BLE connection status, connected device info, battery level, and any active solve session.`,
	RunE:  runStatus,
}

func init() {
	rootCmd.AddCommand(statusCmd)
}

func runStatus(cmd *cobra.Command, args []string) error {
	// Load state file
	stateFile, err := recorder.NewDefaultStateFile()
	if err != nil {
		return fmt.Errorf("failed to load state: %w", err)
	}

	state := stateFile.State()

	fmt.Println("GoCube Solve Recorder Status")
	fmt.Println("============================")
	fmt.Println()

	// Database info
	dbPath := state.DBPath
	if dbPath == "" {
		defaultPath, _ := storage.DefaultDBPath()
		dbPath = defaultPath
	}
	fmt.Printf("Database: %s\n", dbPath)

	// Check if DB exists and get stats
	db, err := storage.Open(dbPath)
	if err == nil {
		defer db.Close()
		if err := db.MigrateUp(); err == nil {
			solveRepo := storage.NewSolveRepository(db)
			solves, _ := solveRepo.List(1)
			if len(solves) > 0 {
				fmt.Printf("Last solve: %s\n", solves[0].StartedAt.Format(time.RFC3339))
			}

			// Count total solves
			allSolves, _ := solveRepo.List(10000)
			fmt.Printf("Total solves: %d\n", len(allSolves))
		}
	}

	fmt.Println()

	// Active solve
	if state.ActiveSolveID != "" {
		fmt.Printf("Active solve: %s\n", state.ActiveSolveID)
		fmt.Println("  (Use 'gocube solve end' to finish or 'gocube solve record' to continue)")
	} else {
		fmt.Println("No active solve")
	}

	fmt.Println()

	// Last device
	if state.LastDeviceID != "" {
		fmt.Printf("Last device: %s (%s)\n", state.LastDeviceName, state.LastDeviceID)
	} else {
		fmt.Println("No device history")
	}

	fmt.Println()

	// Try to scan for devices (uses shared scanning logic)
	_, results, err := ScanForGoCube()
	if err != nil {
		fmt.Printf("Scan error: %v\n", err)
		return nil
	}

	if len(results) == 0 {
		fmt.Println("No GoCube devices found")
		fmt.Println()
		fmt.Println("Tips:")
		fmt.Println("  - Ensure your GoCube is powered on")
		fmt.Println("  - Move the cube to wake it up")
		fmt.Println("  - Check that Bluetooth is enabled on your Mac")
	} else {
		fmt.Printf("Found %d device(s):\n", len(results))
		for _, r := range results {
			fmt.Printf("  - %s (UUID: %s, RSSI: %d)\n", r.Name, r.UUID, r.RSSI)
		}
	}

	return nil
}
