// Package ble provides BLE connectivity for GoCube devices.
package ble

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"tinygo.org/x/bluetooth"

	"github.com/seamusw/gocube/internal/gocube"
)

var (
	ErrNotConnected     = errors.New("not connected to device")
	ErrAlreadyConnected = errors.New("already connected to a device")
	ErrDeviceNotFound   = errors.New("device not found")
	ErrTimeout          = errors.New("connection timeout")
)

// GoCube BLE UUIDs
var (
	serviceUUID = bluetooth.NewUUID(mustParseUUID(gocube.ServiceUUID))
	txCharUUID  = bluetooth.NewUUID(mustParseUUID(gocube.TxCharUUID))
	rxCharUUID  = bluetooth.NewUUID(mustParseUUID(gocube.RxCharUUID))
)

func mustParseUUID(s string) [16]byte {
	var uuid [16]byte
	clean := ""
	for _, c := range s {
		if c != '-' {
			clean += string(c)
		}
	}
	for i := 0; i < 16; i++ {
		var b byte
		fmt.Sscanf(clean[i*2:i*2+2], "%02x", &b)
		uuid[i] = b
	}
	return uuid
}

// Client manages BLE connection to a GoCube device.
type Client struct {
	adapter *bluetooth.Adapter
	device  bluetooth.Device
	txChar  bluetooth.DeviceCharacteristic
	rxChar  bluetooth.DeviceCharacteristic

	mu         sync.RWMutex
	connected  bool
	deviceName string
	deviceUUID string
	battery    int

	onMessage    func(*gocube.Message)
	onDisconnect func()

	autoReconnect bool
	reconnectStop chan struct{}
}

// NewClient creates a new BLE client.
func NewClient() (*Client, error) {
	adapter := bluetooth.DefaultAdapter
	if err := adapter.Enable(); err != nil {
		return nil, fmt.Errorf("failed to enable BLE adapter: %w", err)
	}

	return &Client{
		adapter: adapter,
		battery: -1,
	}, nil
}

// SetMessageCallback sets the callback for incoming messages.
func (c *Client) SetMessageCallback(cb func(*gocube.Message)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.onMessage = cb
}

// SetDisconnectCallback sets the callback for disconnection events.
func (c *Client) SetDisconnectCallback(cb func()) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.onDisconnect = cb
}

// ScanResult represents a discovered GoCube device.
type ScanResult struct {
	Name    string
	UUID    string
	RSSI    int16
	Address bluetooth.Address
}

// Scan scans for GoCube devices.
func (c *Client) Scan(ctx context.Context, timeout time.Duration) ([]ScanResult, error) {
	c.mu.RLock()
	if c.connected {
		c.mu.RUnlock()
		return nil, ErrAlreadyConnected
	}
	c.mu.RUnlock()

	var results []ScanResult
	var mu sync.Mutex
	seen := make(map[string]bool)

	// Channel to signal scan completion
	done := make(chan struct{})

	// Start scan in goroutine since adapter.Scan() blocks
	go func() {
		c.adapter.Scan(func(adapter *bluetooth.Adapter, result bluetooth.ScanResult) {
			name := result.LocalName()
			addr := result.Address.String()

			// Skip if already seen
			mu.Lock()
			if seen[addr] {
				mu.Unlock()
				return
			}
			seen[addr] = true
			mu.Unlock()

			// Check if this is a GoCube by name (case-insensitive prefix match)
			if strings.HasPrefix(strings.ToLower(name), "gocube") {
				mu.Lock()
				results = append(results, ScanResult{
					Name:    name,
					UUID:    addr,
					RSSI:    result.RSSI,
					Address: result.Address,
				})
				mu.Unlock()
			}
		})
		close(done)
	}()

	// Wait for timeout or context cancellation
	select {
	case <-time.After(timeout):
	case <-ctx.Done():
	}

	// Stop scanning
	c.adapter.StopScan()

	// Wait for scan goroutine to finish
	<-done

	return results, nil
}

// Connect connects to a GoCube device by UUID.
func (c *Client) Connect(ctx context.Context, deviceUUID string) error {
	c.mu.Lock()
	if c.connected {
		c.mu.Unlock()
		return ErrAlreadyConnected
	}
	c.mu.Unlock()

	var targetAddr bluetooth.Address
	var targetName string
	found := make(chan struct{})
	var foundOnce sync.Once

	// Start scan in goroutine
	go func() {
		c.adapter.Scan(func(adapter *bluetooth.Adapter, result bluetooth.ScanResult) {
			if result.Address.String() == deviceUUID {
				targetAddr = result.Address
				targetName = result.LocalName()
				foundOnce.Do(func() {
					close(found)
				})
			}
		})
	}()

	// Wait for device or timeout
	select {
	case <-found:
		c.adapter.StopScan()
	case <-time.After(10 * time.Second):
		c.adapter.StopScan()
		return ErrDeviceNotFound
	case <-ctx.Done():
		c.adapter.StopScan()
		return ctx.Err()
	}

	// Connect to the device
	device, err := c.adapter.Connect(targetAddr, bluetooth.ConnectionParams{})
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}

	// Discover services
	services, err := device.DiscoverServices([]bluetooth.UUID{serviceUUID})
	if err != nil {
		device.Disconnect()
		return fmt.Errorf("failed to discover services: %w", err)
	}

	if len(services) == 0 {
		device.Disconnect()
		return fmt.Errorf("GoCube service not found")
	}

	// Discover characteristics
	chars, err := services[0].DiscoverCharacteristics([]bluetooth.UUID{txCharUUID, rxCharUUID})
	if err != nil {
		device.Disconnect()
		return fmt.Errorf("failed to discover characteristics: %w", err)
	}

	var txChar, rxChar bluetooth.DeviceCharacteristic
	for _, ch := range chars {
		if ch.UUID() == txCharUUID {
			txChar = ch
		} else if ch.UUID() == rxCharUUID {
			rxChar = ch
		}
	}

	// Enable notifications on TX characteristic
	err = txChar.EnableNotifications(c.handleNotification)
	if err != nil {
		device.Disconnect()
		return fmt.Errorf("failed to enable notifications: %w", err)
	}

	c.mu.Lock()
	c.device = device
	c.txChar = txChar
	c.rxChar = rxChar
	c.connected = true
	c.deviceName = targetName
	c.deviceUUID = deviceUUID
	c.mu.Unlock()

	// Request initial state
	c.RequestBattery()

	return nil
}

// ConnectFirst connects to the first GoCube device found.
func (c *Client) ConnectFirst(ctx context.Context) error {
	results, err := c.Scan(ctx, 10*time.Second)
	if err != nil {
		return err
	}

	if len(results) == 0 {
		return ErrDeviceNotFound
	}

	return c.ConnectToResult(ctx, results[0])
}

// ConnectToResult connects directly to a device from a scan result.
// This avoids re-scanning and uses the address directly.
func (c *Client) ConnectToResult(ctx context.Context, result ScanResult) error {
	c.mu.Lock()
	if c.connected {
		c.mu.Unlock()
		return ErrAlreadyConnected
	}
	c.mu.Unlock()

	// Connect to the device using the address from scan result
	device, err := c.adapter.Connect(result.Address, bluetooth.ConnectionParams{})
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}

	// Discover services
	services, err := device.DiscoverServices([]bluetooth.UUID{serviceUUID})
	if err != nil {
		device.Disconnect()
		return fmt.Errorf("failed to discover services: %w", err)
	}

	if len(services) == 0 {
		device.Disconnect()
		return fmt.Errorf("GoCube service not found")
	}

	// Discover characteristics
	chars, err := services[0].DiscoverCharacteristics([]bluetooth.UUID{txCharUUID, rxCharUUID})
	if err != nil {
		device.Disconnect()
		return fmt.Errorf("failed to discover characteristics: %w", err)
	}

	var txChar, rxChar bluetooth.DeviceCharacteristic
	for _, ch := range chars {
		if ch.UUID() == txCharUUID {
			txChar = ch
		} else if ch.UUID() == rxCharUUID {
			rxChar = ch
		}
	}

	// Enable notifications on TX characteristic
	err = txChar.EnableNotifications(c.handleNotification)
	if err != nil {
		device.Disconnect()
		return fmt.Errorf("failed to enable notifications: %w", err)
	}

	c.mu.Lock()
	c.device = device
	c.txChar = txChar
	c.rxChar = rxChar
	c.connected = true
	c.deviceName = result.Name
	c.deviceUUID = result.UUID
	c.mu.Unlock()

	// Request initial state
	c.RequestBattery()

	return nil
}

// Disconnect disconnects from the current device.
func (c *Client) Disconnect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.connected {
		return nil
	}

	err := c.device.Disconnect()
	c.connected = false
	c.deviceName = ""
	c.deviceUUID = ""
	c.battery = -1

	return err
}

// IsConnected returns true if connected to a device.
func (c *Client) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.connected
}

// DeviceName returns the connected device name.
func (c *Client) DeviceName() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.deviceName
}

// DeviceUUID returns the connected device UUID.
func (c *Client) DeviceUUID() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.deviceUUID
}

// Battery returns the last known battery level (-1 if unknown).
func (c *Client) Battery() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.battery
}

// SendCommand sends a command to the cube.
func (c *Client) SendCommand(cmd byte) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.connected {
		return ErrNotConnected
	}

	data := gocube.BuildCommand(cmd)
	// Try WriteWithoutResponse first, fall back to regular Write
	_, err := c.rxChar.WriteWithoutResponse(data)
	if err != nil {
		// Some BLE stacks require regular Write
		_, err = c.rxChar.Write(data)
	}
	return err
}

// RequestBattery requests the battery level from the cube.
func (c *Client) RequestBattery() error {
	return c.SendCommand(gocube.CmdRequestBattery)
}

// RequestState requests the current cube state.
func (c *Client) RequestState() error {
	return c.SendCommand(gocube.CmdRequestState)
}

// FlashBacklight flashes the cube backlight three times.
func (c *Client) FlashBacklight() error {
	return c.SendCommand(gocube.CmdFlashBacklight)
}

// SlowFlashBacklight slowly flashes the cube backlight three times.
func (c *Client) SlowFlashBacklight() error {
	return c.SendCommand(gocube.CmdSlowFlashBacklight)
}

// ToggleBacklight toggles the cube backlight on/off.
func (c *Client) ToggleBacklight() error {
	return c.SendCommand(gocube.CmdToggleBacklight)
}

// ToggleAnimatedBacklight enables/disables animated backlight.
func (c *Client) ToggleAnimatedBacklight() error {
	return c.SendCommand(gocube.CmdToggleAnimatedBL)
}

// EnableOrientation enables orientation tracking on the cube.
func (c *Client) EnableOrientation() error {
	return c.SendCommand(gocube.CmdEnableOrientation)
}

// DisableOrientation disables orientation tracking on the cube.
func (c *Client) DisableOrientation() error {
	return c.SendCommand(gocube.CmdDisableOrientation)
}

// CalibrateOrientation calibrates the cube's orientation sensor.
func (c *Client) CalibrateOrientation() error {
	return c.SendCommand(gocube.CmdCalibrateOrientation)
}

// handleNotification handles incoming BLE notifications.
func (c *Client) handleNotification(data []byte) {
	msg, err := gocube.ParseMessage(data)
	if err != nil {
		return
	}

	// Handle battery updates internally
	if msg.Type == gocube.MsgTypeBattery {
		if battery, err := gocube.DecodeBattery(msg.Payload); err == nil {
			c.mu.Lock()
			c.battery = battery.Level
			c.mu.Unlock()
		}
	}

	// Call user callback
	c.mu.RLock()
	cb := c.onMessage
	c.mu.RUnlock()

	if cb != nil {
		cb(msg)
	}
}

// EnableAutoReconnect enables automatic reconnection on disconnect.
func (c *Client) EnableAutoReconnect(enable bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.autoReconnect = enable
}
