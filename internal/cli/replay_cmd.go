package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/seamusw/gocube/internal/cube"
	"github.com/seamusw/gocube/internal/gocube"
	"github.com/seamusw/gocube/pkg/types"
)

var replayCmd = &cobra.Command{
	Use:   "replay [log-file]",
	Short: "Replay a recorded solve session",
	Long: `Replay a previously recorded solve session from a log file.
This allows you to debug phase detection issues without needing the physical cube.

If no log file is specified, lists available log files.

Usage:
  gocube solve replay                    # List available logs
  gocube solve replay <log-file>         # Replay specific log
  gocube solve replay --speed 2.0        # Replay at 2x speed
  gocube solve replay --step             # Step through events manually`,
	RunE: runReplay,
}

var (
	replaySpeed float64
	replayStep  bool
)

func init() {
	solveCmd.AddCommand(replayCmd)
	replayCmd.Flags().Float64VarP(&replaySpeed, "speed", "s", 1.0, "Playback speed multiplier")
	replayCmd.Flags().BoolVarP(&replayStep, "step", "t", false, "Step through events manually")
}

func runReplay(cmd *cobra.Command, args []string) error {
	homeDir, _ := os.UserHomeDir()
	logDir := filepath.Join(homeDir, ".gocube_recorder", "logs")

	// If no args, list available logs
	if len(args) == 0 {
		return listLogs(logDir)
	}

	logPath := args[0]
	// If not an absolute path, look in log directory
	if !filepath.IsAbs(logPath) {
		logPath = filepath.Join(logDir, logPath)
	}

	// Load the log
	log, err := LoadSolveLog(logPath)
	if err != nil {
		return fmt.Errorf("failed to load log: %w", err)
	}

	fmt.Printf("Loaded log: %s\n", logPath)
	fmt.Printf("Created: %s\n", log.CreatedAt.Format(time.RFC3339))
	fmt.Printf("Events: %d\n", len(log.Events))
	fmt.Println()

	// Create replay model
	model := newReplayModel(log, replaySpeed, replayStep)
	p := tea.NewProgram(model, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		return fmt.Errorf("replay error: %w", err)
	}

	return nil
}

func listLogs(logDir string) error {
	entries, err := os.ReadDir(logDir)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("No log files found. Record a solve first with: gocube solve record")
			return nil
		}
		return err
	}

	var logs []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".jsonl") {
			logs = append(logs, e.Name())
		}
	}

	if len(logs) == 0 {
		fmt.Println("No log files found. Record a solve first with: gocube solve record")
		return nil
	}

	// Sort by name (which includes timestamp, so newest last)
	sort.Strings(logs)

	fmt.Println("Available log files:")
	fmt.Println()
	for _, log := range logs {
		fmt.Printf("  %s\n", log)
	}
	fmt.Println()
	fmt.Println("Usage: gocube solve replay <filename>")

	return nil
}

// Replay model
type replayModel struct {
	log          *SolveLog
	eventIndex   int
	speed        float64
	stepMode     bool
	paused       bool
	tracker      *cube.Tracker
	moves        []types.Move
	currentPhase string
	detectedPhase string
	startTime    time.Time
	elapsed      time.Duration
	lastEventTime int64
	quitting     bool
	debugMode    bool
}

func newReplayModel(log *SolveLog, speed float64, stepMode bool) *replayModel {
	return &replayModel{
		log:       log,
		speed:     speed,
		stepMode:  stepMode,
		paused:    stepMode, // Start paused in step mode
		tracker:   cube.NewTracker(),
		moves:     make([]types.Move, 0),
		startTime: time.Now(),
	}
}

type replayTickMsg time.Time
type replayEventMsg struct{ event LogEvent }

func (m *replayModel) Init() tea.Cmd {
	if m.stepMode {
		return nil // Wait for user input in step mode
	}
	return m.scheduleNextEvent()
}

func (m *replayModel) scheduleNextEvent() tea.Cmd {
	if m.eventIndex >= len(m.log.Events) {
		return nil
	}

	event := m.log.Events[m.eventIndex]

	// Calculate delay based on elapsed time difference
	var delay time.Duration
	if m.lastEventTime > 0 {
		delayMs := event.ElapsedMs - m.lastEventTime
		delay = time.Duration(float64(delayMs) / m.speed) * time.Millisecond
	}

	return tea.Tick(delay, func(t time.Time) tea.Msg {
		return replayEventMsg{event: event}
	})
}

func (m *replayModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			m.quitting = true
			return m, tea.Quit

		case " ", "n":
			if m.stepMode || m.paused {
				// Advance to next event
				if m.eventIndex < len(m.log.Events) {
					event := m.log.Events[m.eventIndex]
					m.processEvent(event)
					m.eventIndex++
				}
			} else {
				// Toggle pause
				m.paused = !m.paused
				if !m.paused {
					return m, m.scheduleNextEvent()
				}
			}

		case "p":
			m.paused = !m.paused
			if !m.paused && !m.stepMode {
				return m, m.scheduleNextEvent()
			}

		case "r":
			// Reset replay
			m.eventIndex = 0
			m.tracker.Reset()
			m.moves = nil
			m.currentPhase = ""
			m.detectedPhase = ""
			m.lastEventTime = 0
			m.startTime = time.Now()

		case "d":
			m.debugMode = !m.debugMode

		case "+", "=":
			m.speed *= 2
			if m.speed > 16 {
				m.speed = 16
			}

		case "-":
			m.speed /= 2
			if m.speed < 0.25 {
				m.speed = 0.25
			}
		}

	case replayEventMsg:
		if !m.paused {
			m.processEvent(msg.event)
			m.eventIndex++
			return m, m.scheduleNextEvent()
		}
	}

	return m, nil
}

func (m *replayModel) processEvent(event LogEvent) {
	m.lastEventTime = event.ElapsedMs
	m.elapsed = time.Duration(event.ElapsedMs) * time.Millisecond

	switch event.EventType {
	case LogEventBLEMessage:
		if event.BLEType == gocube.MsgTypeRotation {
			rotations, err := gocube.DecodeRotation(event.BLEPayload)
			if err == nil {
				moves := gocube.RotationsToMoves(rotations, 0)
				for _, move := range moves {
					m.moves = append(m.moves, move)
					prevHighest := m.tracker.HighestPhaseKey()
					m.tracker.ApplyMove(move)
					newPhase := m.tracker.CurrentPhaseKey()
					newHighest := m.tracker.HighestPhaseKey()
					m.detectedPhase = newPhase

					// Only update current phase when reaching a NEW highest (monotonic)
					if newHighest != prevHighest && newHighest != "scrambled" {
						m.currentPhase = newHighest
					}
				}
			}
		}

	case LogEventKeyPress:
		// Key presses are informational in replay

	case LogEventPhase:
		// Phase changes are recorded but we recalculate them
	}
}

func (m *replayModel) View() string {
	if m.quitting {
		return "Replay ended.\n"
	}

	var b strings.Builder

	// Title
	b.WriteString(titleStyle.Render("GoCube Solve Replay"))
	b.WriteString("\n\n")

	// Replay status
	progress := fmt.Sprintf("Event %d/%d", m.eventIndex, len(m.log.Events))
	if m.paused {
		progress += " [PAUSED]"
	}
	if m.stepMode {
		progress += " [STEP MODE]"
	}
	b.WriteString(statusStyle.Render(progress))
	b.WriteString(fmt.Sprintf(" (%.1fx speed)\n", m.speed))

	// Elapsed time
	b.WriteString(fmt.Sprintf("Time: %s\n", m.formatElapsed()))
	b.WriteString("\n")

	// Phase detection (monotonic - never goes backwards)
	if m.tracker != nil {
		if m.tracker.IsSolved() {
			b.WriteString(fmt.Sprintf("Cube State: %s\n", phaseStyle.Render("SOLVED!")))
		} else {
			// Show the NEXT phase to work on based on highest phase reached (monotonic)
			highestPhase := m.tracker.HighestPhaseKey()
			workingOn := getNextPhaseFromHighest(highestPhase)
			b.WriteString(fmt.Sprintf("Working on: %s\n", phaseStyle.Render(workingOn)))
			// Show last completed phase
			if highestPhase != "scrambled" && highestPhase != "inspection" {
				b.WriteString(fmt.Sprintf("Completed: %s\n", statusStyle.Render(phaseDisplayName(highestPhase))))
			}
		}
	}

	// Move count
	b.WriteString(fmt.Sprintf("Moves: %d\n", len(m.moves)))
	b.WriteString("\n")

	// Recent moves
	if len(m.moves) > 0 {
		b.WriteString("Moves: ")
		start := 0
		if len(m.moves) > 20 {
			start = len(m.moves) - 20
			b.WriteString("... ")
		}
		var notations []string
		for i := start; i < len(m.moves); i++ {
			notations = append(notations, m.moves[i].Notation())
		}
		b.WriteString(moveStyle.Render(strings.Join(notations, " ")))
		b.WriteString("\n")
	}

	// Debug mode: show cube state
	if m.debugMode && m.tracker != nil {
		b.WriteString("\n")
		b.WriteString(statusStyle.Render("DEBUG - Cube State:"))
		b.WriteString("\n")
		b.WriteString(m.tracker.CubeString())
	}

	// Current event info
	if m.eventIndex < len(m.log.Events) {
		event := m.log.Events[m.eventIndex]
		b.WriteString("\n")
		b.WriteString(statusStyle.Render(fmt.Sprintf("Next: %s - %s", event.EventType, event.Description)))
		b.WriteString("\n")
	}

	b.WriteString("\n")

	// Help
	help := "SPACE/n=next  p=pause  r=reset  d=debug  +/-=speed  q=quit"
	if m.stepMode {
		help = "SPACE/n=next event  r=reset  d=debug  q=quit"
	}
	b.WriteString(helpStyle.Render(help))
	b.WriteString("\n")

	return b.String()
}

func (m *replayModel) formatElapsed() string {
	if m.elapsed < time.Minute {
		return fmt.Sprintf("%.1fs", m.elapsed.Seconds())
	}
	mins := int(m.elapsed.Minutes())
	secs := m.elapsed.Seconds() - float64(mins*60)
	return fmt.Sprintf("%d:%05.2f", mins, secs)
}
