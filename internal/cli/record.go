package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	"github.com/SeamusWaldron/gocube"
	"github.com/SeamusWaldron/gocube/internal/recorder"
	"github.com/SeamusWaldron/gocube/internal/storage"
)

var recordCmd = &cobra.Command{
	Use:   "record",
	Short: "Interactive solve recording mode",
	Long: `Start an interactive TUI for recording solves with real-time move display
and keyboard shortcuts for phase marking.

Keyboard shortcuts:
  s       - Start a new solve
  e       - End the current solve
  1-6     - Mark phase (1=inspection, 2=white_cross, 3=white_corners,
            4=middle_layer, 5=bottom_perm, 6=bottom_orient)
  q/Esc   - Quit

The TUI will display moves in real-time as you solve the cube.`,
	RunE: runRecord,
}

func init() {
	solveCmd.AddCommand(recordCmd)
}

// Styles
var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("205"))

	statusStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	phaseStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("39"))

	moveStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("82"))

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196"))

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))
)

// Messages
type tickMsg time.Time
type bleConnectedMsg struct{ name string }
type bleDisconnectedMsg struct{}
type bleMessageMsg struct{ msg *gocube.Message }
type moveRecordedMsg struct{ move gocube.Move }
type phaseMarkedMsg struct{ phase string }
type inspectionFlashMsg struct{} // Periodic flash during inspection
type solvedLedOffMsg struct{}    // Turn LED off after solve celebration

// Messages for auto-detected phase changes
type phaseDetectedMsg struct{ phase string }

// Model
type recordModel struct {
	// BLE
	client       *gocube.Client
	connected    bool
	deviceName   string
	battery      int
	msgChan      chan *gocube.Message
	scanResults  []gocube.ScanResult // Pre-scanned devices
	prescanClient *gocube.Client      // Client used for pre-scan

	// Database
	db        *storage.DB
	stateFile *recorder.StateFile
	session   *recorder.Session

	// Cube state tracking
	tracker       *gocube.Tracker
	autoPhase     bool   // whether to auto-detect phases
	detectedPhase string // current detected phase from cube state
	solveStarted  bool   // true once first move is made after inspection
	inspecting    bool   // true after SPACE pressed, waiting for first move
	debugMode     bool   // show detailed cube state for debugging

	// Timing
	inspectStart  time.Time // when inspection started (SPACE pressed)

	// State
	recording    bool
	solveID      string
	currentPhase string
	moves        []gocube.Move
	startTime    time.Time
	elapsed      time.Duration

	// UI
	width     int
	height    int
	err       error
	quitting  bool

	// Logging
	logger    *SolveLogger
	logPath   string
}

func newRecordModel(db *storage.DB, stateFile *recorder.StateFile, prescanClient *gocube.Client, scanResults []gocube.ScanResult) *recordModel {
	// Create logger and start logging
	logger := NewSolveLogger()
	homeDir, _ := os.UserHomeDir()
	logDir := filepath.Join(homeDir, ".gocube_recorder", "logs")
	if err := logger.Start(logDir); err != nil {
		// Log error but continue - logging is optional
		fmt.Printf("Warning: could not start logging: %v\n", err)
	}

	return &recordModel{
		db:            db,
		stateFile:     stateFile,
		session:       recorder.NewSession(db, stateFile),
		tracker:       gocube.NewTracker(),
		autoPhase:     true, // Enable auto phase detection
		battery:       -1,
		msgChan:       make(chan *gocube.Message, 100),
		prescanClient: prescanClient,
		scanResults:   scanResults,
		logger:        logger,
	}
}

func (m *recordModel) Init() tea.Cmd {
	return tea.Batch(
		m.connectBLE(),
		m.tickCmd(),
		m.listenForMessages(),
	)
}

func (m *recordModel) listenForMessages() tea.Cmd {
	return func() tea.Msg {
		msg := <-m.msgChan
		return bleMessageMsg{msg: msg}
	}
}

func (m *recordModel) tickCmd() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// scheduleInspectionFlash schedules the next inspection flash (every 3 seconds)
func (m *recordModel) scheduleInspectionFlash() tea.Cmd {
	return tea.Tick(3*time.Second, func(t time.Time) tea.Msg {
		return inspectionFlashMsg{}
	})
}

// scheduleSolvedLedOff schedules turning off the LED after solve celebration
func (m *recordModel) scheduleSolvedLedOff() tea.Cmd {
	return tea.Tick(5*time.Second, func(t time.Time) tea.Msg {
		return solvedLedOffMsg{}
	})
}

func (m *recordModel) connectBLE() tea.Cmd {
	return func() tea.Msg {
		// Must have prescan client and results - no scanning in TUI
		if m.prescanClient == nil || len(m.scanResults) == 0 {
			m.err = fmt.Errorf("no device found in pre-scan")
			return nil
		}

		client := m.prescanClient
		m.client = client

		// Set up message callback BEFORE connecting
		client.SetMessageCallback(func(msg *gocube.Message) {
			// Send to channel for BubbleTea to pick up
			select {
			case m.msgChan <- msg:
			default:
				// Channel full, drop message
			}
		})

		ctx := context.Background()
		state := m.stateFile.State()
		results := m.scanResults

		// Find the target device - prefer last known device if found in scan
		var target *gocube.ScanResult
		if state.LastDeviceID != "" {
			for i := range results {
				if results[i].UUID == state.LastDeviceID {
					target = &results[i]
					break
				}
			}
		}
		// If last device not in scan results, use first found
		if target == nil {
			target = &results[0]
		}

		// Connect directly using the scan result (no re-scan needed)
		if err := client.ConnectToResult(ctx, *target); err != nil {
			m.err = fmt.Errorf("connection failed: %w", err)
			return nil
		}

		// Enable orientation tracking for cube rotation detection
		if err := client.EnableOrientation(); err != nil {
			// Log but don't fail - orientation is optional
		}

		return bleConnectedMsg{name: client.DeviceName()}
	}
}

func (m *recordModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Log all key presses
		if m.logger != nil {
			m.logger.LogKeyPress(msg.String())
		}

		switch msg.String() {
		case "q", "esc", "ctrl+c":
			m.quitting = true
			if m.client != nil {
				m.client.Disconnect()
			}
			if m.logger != nil {
				m.logPath = m.logger.FilePath()
				m.logger.Close()
			}
			return m, tea.Quit

		case "s":
			if !m.recording {
				return m, m.startSolve()
			}

		case "e":
			if m.recording {
				return m, m.endSolve()
			}

		case "0", "1", "2", "3", "4", "5", "6", "7":
			if m.recording {
				num := int(msg.String()[0] - '0')
				phase := storage.NumberToPhaseKey(num)
				if phase != "" {
					return m, m.markPhase(phase)
				}
			}

		case "r", "l":
			if m.recording {
				phase := storage.AlgoKeyToPhaseKey(msg.String())
				if phase != "" {
					return m, m.markPhase(phase)
				}
			}

		case "d":
			// Toggle debug mode
			m.debugMode = !m.debugMode

		case " ", "enter":
			// SPACE/ENTER ends scramble, starts inspection (before first move)
			if m.recording && !m.solveStarted && !m.inspecting {
				m.inspecting = true
				m.inspectStart = time.Now()
				m.currentPhase = "inspection"

				// Mark inspection phase
				if m.autoPhase {
					if err := m.session.MarkPhase("inspection", nil); err != nil {
						m.err = err
					}
				}

				// Log the transition
				if m.logger != nil {
					m.logger.LogKeyPress(" ")
				}

				// Start slow flash during inspection and schedule repeating flash
				if m.client != nil {
					m.client.SlowFlashBacklight()
				}
				return m, m.scheduleInspectionFlash()
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tickMsg:
		// Only update elapsed time after solve has started (not during scramble/inspection)
		if m.recording && m.solveStarted {
			m.elapsed = time.Since(m.startTime)
		}
		if m.client != nil {
			m.battery = m.client.Battery()
		}
		return m, m.tickCmd()

	case bleConnectedMsg:
		m.connected = true
		m.deviceName = msg.name
		if m.stateFile != nil && m.client != nil {
			m.stateFile.SetLastDevice(m.client.DeviceUUID(), m.deviceName)
		}
		// Flash LED on connect (with slight delay for BLE stack to settle)
		if m.client != nil {
			go func() {
				time.Sleep(500 * time.Millisecond)
				m.client.FlashBacklight()
			}()
		}

	case bleDisconnectedMsg:
		m.connected = false
		m.deviceName = ""

	case inspectionFlashMsg:
		// Repeat slow flash while still in inspection mode
		if m.inspecting && !m.solveStarted && m.client != nil {
			m.client.SlowFlashBacklight()
			return m, m.scheduleInspectionFlash()
		}

	case solvedLedOffMsg:
		// Turn off LED after solve celebration
		if m.client != nil {
			m.client.ToggleBacklight()
		}

	case bleMessageMsg:
		// Log all BLE messages
		if m.logger != nil {
			desc := gocube.MessageTypeName(msg.msg.Type)
			if msg.msg.Type == gocube.MsgTypeRotation {
				if rotations, err := gocube.DecodeRotation(msg.msg.Payload); err == nil {
					moves := gocube.RotationsToMoves(rotations, 0)
					var notations []string
					for _, mv := range moves {
						notations = append(notations, mv.Notation())
					}
					desc = fmt.Sprintf("rotation: %s", strings.Join(notations, " "))
				}
			}
			m.logger.LogBLEMessage(msg.msg, desc)
		}

		// Check if this is the first move after inspection - mark phase BEFORE recording
		if m.recording && m.inspecting && !m.solveStarted && msg.msg.Type == gocube.MsgTypeRotation {
			m.solveStarted = true
			m.inspecting = false
			m.startTime = time.Now()
			m.elapsed = 0

			// Mark white_cross with a timestamp 1ms BEFORE the move will be recorded.
			// This ensures the move falls into white_cross phase, not inspection.
			// HandleMessage calculates its own timestamp, which will be >= currentTs.
			if m.autoPhase {
				currentTs := m.session.CurrentTimestamp()
				phaseTs := currentTs - 1
				if phaseTs < 0 {
					phaseTs = 0
				}
				if err := m.session.MarkPhaseAt("white_cross", phaseTs, nil); err != nil {
					m.err = fmt.Errorf("failed to mark white_cross: %w", err)
				} else {
					m.currentPhase = "white_cross"
					if m.logger != nil {
						m.logger.LogPhaseChange("white_cross")
					}
				}
			}
		}

		// Process the BLE message through the session
		if m.recording && m.session != nil {
			if err := m.session.HandleMessage(msg.msg); err != nil {
				m.err = err
			}
			// Decode moves directly here to avoid goroutine race
			if msg.msg.Type == gocube.MsgTypeRotation {
				if rotations, err := gocube.DecodeRotation(msg.msg.Payload); err == nil {
					moves := gocube.RotationsToMoves(rotations, 0)
					for _, move := range moves {
						m.moves = append(m.moves, move)

						// Update cube tracker
						if m.tracker != nil {
							prevHighest := m.tracker.HighestPhaseKey()
							m.tracker.ApplyMove(move)
							newPhase := m.tracker.CurrentPhaseKey()
							newHighest := m.tracker.HighestPhaseKey()

							// Update detected phase display (shows current cube state)
							m.detectedPhase = newPhase

							// Handle phase transitions - only after solve started
							// Only mark when reaching a NEW highest phase (monotonic progression)
							// Skip: scrambled (not a real phase), white_cross (marked at solve start),
							// and complete (just a state indicator, not a phase with moves)
							if m.autoPhase && m.solveStarted && newHighest != prevHighest &&
								newHighest != "scrambled" && newHighest != "white_cross" && newHighest != "complete" {
								// Auto-mark phase completions during solving
								if err := m.session.MarkPhase(newHighest, nil); err == nil {
									m.currentPhase = newHighest
									// Log phase change
									if m.logger != nil {
										m.logger.LogPhaseChange(newHighest)
									}
									// Flash LED on phase complete
									if m.client != nil {
										m.client.ToggleBacklight()
									}
								}
							}

							// Auto-end solve when completed
							if m.solveStarted && m.tracker.IsSolved() {
								m.session.End()
								m.recording = false
								m.currentPhase = "complete"

								// LED celebration: turn on for 5 seconds
								if m.client != nil {
									m.client.ToggleBacklight()
								}
								return m, tea.Batch(
									m.listenForMessages(),
									m.scheduleSolvedLedOff(),
								)
							}
						}
					}
				}
			}
		}
		// Continue listening for more messages
		return m, m.listenForMessages()

	case moveRecordedMsg:
		m.moves = append(m.moves, msg.move)

	case phaseMarkedMsg:
		m.currentPhase = msg.phase

	case phaseDetectedMsg:
		m.detectedPhase = msg.phase
	}

	return m, nil
}

func (m *recordModel) startSolve() tea.Cmd {
	return func() tea.Msg {
		deviceName := ""
		deviceID := ""
		if m.client != nil {
			deviceName = m.client.DeviceName()
			deviceID = m.client.DeviceUUID()
		}

		solveID, err := m.session.Start("", "", deviceName, deviceID, "0.1.0")
		if err != nil {
			m.err = err
			return nil
		}

		m.solveID = solveID
		m.recording = true
		m.startTime = time.Now()
		m.moves = nil
		m.currentPhase = "scramble"
		m.detectedPhase = "complete" // Start assumes solved cube
		m.solveStarted = false       // User must press SPACE after scrambling
		m.inspecting = false         // Not yet in inspection

		// Reset tracker to solved state
		if m.tracker != nil {
			m.tracker.Reset()
		}

		// Auto-mark scramble phase at start
		if m.autoPhase {
			if err := m.session.MarkPhase("scramble", nil); err != nil {
				m.err = err
			}
		}

		return nil
	}
}

func (m *recordModel) endSolve() tea.Cmd {
	return func() tea.Msg {
		if err := m.session.End(); err != nil {
			m.err = err
			return nil
		}

		m.recording = false
		return nil
	}
}

func (m *recordModel) markPhase(phase string) tea.Cmd {
	return func() tea.Msg {
		if err := m.session.MarkPhase(phase, nil); err != nil {
			m.err = err
			return nil
		}
		return phaseMarkedMsg{phase: phase}
	}
}

func (m *recordModel) View() string {
	if m.quitting {
		msg := "Goodbye!\n"
		if m.logPath != "" {
			msg += fmt.Sprintf("Log saved to: %s\n", m.logPath)
		}
		return msg
	}

	var b strings.Builder

	// Title
	b.WriteString(titleStyle.Render("GoCube Solve Recorder"))
	b.WriteString("\n\n")

	// Connection status
	if m.connected {
		status := fmt.Sprintf("Connected: %s", m.deviceName)
		if m.battery >= 0 {
			status += fmt.Sprintf(" (Battery: %d%%)", m.battery)
		}
		b.WriteString(statusStyle.Render(status))
	} else if len(m.scanResults) == 0 {
		b.WriteString(errorStyle.Render("No device found - run again to retry"))
	} else {
		b.WriteString(errorStyle.Render("Connecting..."))
	}
	b.WriteString("\n\n")

	// Recording status
	if m.recording {
		b.WriteString(phaseStyle.Render(fmt.Sprintf("RECORDING: %s", m.formatElapsed())))
		b.WriteString("\n")
		b.WriteString(fmt.Sprintf("Solve ID: %s\n", m.solveID[:8]))

		// Show current workflow state
		if !m.solveStarted {
			if m.inspecting {
				// After SPACE, waiting for first move
				b.WriteString(fmt.Sprintf("State: %s - make first move to start timer\n", phaseStyle.Render("INSPECTION")))
			} else if m.tracker != nil && m.tracker.IsSolved() {
				// Still scrambling
				b.WriteString(fmt.Sprintf("State: %s\n", phaseStyle.Render("SCRAMBLE THE CUBE")))
			} else {
				// Cube is scrambled, ready for SPACE
				b.WriteString(fmt.Sprintf("State: %s - press SPACE when ready\n", phaseStyle.Render("READY")))
			}
		} else {
			// Solving - show current working phase (monotonic, never goes backwards)
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
		}

		// Debug mode: show cube state
		if m.debugMode && m.tracker != nil {
			b.WriteString("\n")
			b.WriteString(statusStyle.Render("DEBUG - Cube State:"))
			b.WriteString("\n")
			b.WriteString(m.tracker.CubeString())
		}

		// Show last completed phase (only if we've completed at least one phase)
		if m.currentPhase != "" && m.currentPhase != "inspection" {
			b.WriteString(fmt.Sprintf("Last completed: %s\n", statusStyle.Render(phaseDisplayName(m.currentPhase))))
		}

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
	} else {
		if m.solveID != "" {
			// Just finished
			b.WriteString(fmt.Sprintf("Solve complete: %s\n", m.solveID))
			b.WriteString(fmt.Sprintf("Duration: %s\n", m.formatElapsed()))
			b.WriteString(fmt.Sprintf("Total moves: %d\n", len(m.moves)))
			if m.elapsed.Seconds() > 0 {
				tps := float64(len(m.moves)) / m.elapsed.Seconds()
				b.WriteString(fmt.Sprintf("TPS: %.2f\n", tps))
			}
			b.WriteString("\n")
			b.WriteString("Press 's' to start a new solve (cube must be SOLVED first)\n")
		} else {
			b.WriteString("Ready to record\n")
			b.WriteString("Press 's' to start (cube must be SOLVED first)\n")
		}
	}

	// Error
	if m.err != nil {
		b.WriteString("\n")
		b.WriteString(errorStyle.Render(fmt.Sprintf("Error: %v", m.err)))
		b.WriteString("\n")
	}

	b.WriteString("\n")

	// Help
	help := "Keys: s=start  d=debug  q=quit"
	if m.recording {
		if !m.solveStarted {
			help = "Scramble cube, then SPACE=start solve | d=debug e=end q=quit"
		} else {
			help = "Phases: 1-7 | r=RHS l=LHS | d=debug e=end q=quit"
		}
	}
	b.WriteString(helpStyle.Render(help))
	b.WriteString("\n")

	return b.String()
}

func (m *recordModel) formatElapsed() string {
	if m.elapsed < time.Minute {
		return fmt.Sprintf("%.1fs", m.elapsed.Seconds())
	}
	mins := int(m.elapsed.Minutes())
	secs := m.elapsed.Seconds() - float64(mins*60)
	return fmt.Sprintf("%d:%05.2f", mins, secs)
}

func phaseDisplayName(key string) string {
	return storage.PhaseDisplayName(key)
}

// getNextPhase returns the name of the next phase to work on based on progress
func getNextPhase(progress gocube.PhaseProgress) string {
	if !progress.WhiteCross {
		return "White Cross"
	}
	if !progress.TopLayer {
		return "Top Corners"
	}
	if !progress.MiddleLayer {
		return "Middle Layer"
	}
	if !progress.BottomCross {
		return "Bottom Cross"
	}
	if !progress.CornersPositioned {
		return "Position Corners"
	}
	if !progress.CornersOriented {
		return "Orient Corners"
	}
	return "Complete"
}

// getNextPhaseFromHighest returns the next phase to work on based on highest completed phase
// This is monotonic - it never goes backwards
func getNextPhaseFromHighest(highestPhase string) string {
	switch highestPhase {
	case "scrambled", "scramble", "inspection":
		return "White Cross"
	case "white_cross":
		return "Top Corners"
	case "top_corners":
		return "Middle Layer"
	case "middle_layer":
		return "Bottom Cross"
	case "bottom_cross":
		return "Position Corners"
	case "position_corners":
		return "Orient Corners"
	case "orient_corners", "complete":
		return "Complete"
	default:
		return "White Cross"
	}
}

func runRecord(cmd *cobra.Command, args []string) error {
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

	// Pre-scan for GoCube devices BEFORE starting TUI
	// Uses the same scanning logic as 'gocube status'
	prescanClient, scanResults, err := ScanForGoCube()
	if err != nil {
		return err
	}

	if len(scanResults) == 0 {
		fmt.Println("No GoCube devices found.")
		fmt.Println()
		fmt.Println("To fix this:")
		fmt.Println("  1. Rotate your cube to wake it up")
		fmt.Println("  2. Make sure it's not connected to your phone")
		fmt.Println("  3. Run this command again")
		return nil // Exit without entering TUI
	}

	// Check for existing active solve
	if stateFile.HasActiveSolve() {
		fmt.Printf("Resuming active solve: %s\n", stateFile.ActiveSolveID())
	}

	model := newRecordModel(db, stateFile, prescanClient, scanResults)
	p := tea.NewProgram(model, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		return fmt.Errorf("TUI error: %w", err)
	}

	return nil
}
