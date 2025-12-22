// Package cli implements the command-line interface for gocube.
package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

const version = "0.1.0"

var (
	// Global flags
	dbPath  string
	verbose bool
)

// rootCmd is the base command.
var rootCmd = &cobra.Command{
	Use:   "gocube",
	Short: "GoCube Solve Recorder",
	Long: `GoCube Solve Recorder - A CLI tool for recording and analyzing Rubik's Cube solves
using a GoCube smart cube.

Connect to your GoCube over Bluetooth, record solves with phase marking,
and generate detailed analysis reports to improve your solving technique.`,
	Version: version,
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&dbPath, "db", "", "Database file path (default: ~/.gocube_recorder/gocube.db)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")
}

// getDBPath returns the database path from flag or default.
func getDBPath() string {
	if dbPath != "" {
		return dbPath
	}
	return "" // Will use default
}
