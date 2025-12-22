package gocube

// Option configures GoCube behavior.
type Option func(*config)

type config struct {
	autoReconnect  bool
	moveHistory    bool
	phaseDetection bool
}

func defaultConfig() *config {
	return &config{
		autoReconnect:  false,
		moveHistory:    true,
		phaseDetection: true,
	}
}

// WithAutoReconnect enables automatic reconnection on disconnect.
// When enabled, the GoCube will attempt to reconnect if the connection drops.
func WithAutoReconnect(enabled bool) Option {
	return func(c *config) {
		c.autoReconnect = enabled
	}
}

// WithMoveHistory enables or disables move history tracking.
// When enabled (default), all moves are stored and accessible via Moves().
// Disable this for long sessions to reduce memory usage.
func WithMoveHistory(enabled bool) Option {
	return func(c *config) {
		c.moveHistory = enabled
	}
}

// WithPhaseDetection enables or disables automatic phase detection.
// When enabled (default), the OnPhaseChange callback fires when phases complete.
func WithPhaseDetection(enabled bool) Option {
	return func(c *config) {
		c.phaseDetection = enabled
	}
}
