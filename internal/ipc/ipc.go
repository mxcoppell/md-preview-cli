package ipc

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"time"
)

// ErrHostAlreadyRunning is returned by NewServer when a live host is already
// listening on the socket.
var ErrHostAlreadyRunning = errors.New("host already running")

// OpenRequest is sent from CLI to host to open a new window.
type OpenRequest struct {
	ConfigPath string `json:"config_path"`
}

// OpenResponse is sent from host back to CLI.
type OpenResponse struct {
	OK       bool   `json:"ok"`
	WindowID string `json:"window_id,omitempty"`
	Error    string `json:"error,omitempty"`
	Reused   bool   `json:"reused,omitempty"`
}

// SocketPath returns the Unix socket path for IPC.
// The filename includes the UID to prevent collisions between users sharing /tmp.
func SocketPath() string {
	return filepath.Join(os.TempDir(), fmt.Sprintf("md-preview-cli-%d.sock", os.Getuid()))
}

// Dial connects to the host process with a timeout.
func Dial() (net.Conn, error) {
	return net.DialTimeout("unix", SocketPath(), 500*time.Millisecond)
}

// IsHostRunning checks if a host process is listening on the socket.
func IsHostRunning() bool {
	conn, err := Dial()
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

// CleanStaleSocket removes the socket file if no host is listening.
func CleanStaleSocket() {
	path := SocketPath()
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return
	}
	if !IsHostRunning() {
		os.Remove(path)
	}
}

// SendOpen sends an open request to the host and reads the response.
func SendOpen(conn net.Conn, cfgPath string) (OpenResponse, error) {
	req := OpenRequest{ConfigPath: cfgPath}
	data, err := json.Marshal(req)
	if err != nil {
		return OpenResponse{}, fmt.Errorf("marshal request: %w", err)
	}
	data = append(data, '\n')

	conn.SetWriteDeadline(time.Now().Add(2 * time.Second))
	if _, err := conn.Write(data); err != nil {
		return OpenResponse{}, fmt.Errorf("write request: %w", err)
	}

	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	buf := make([]byte, 4096)
	n, err := conn.Read(buf)
	if err != nil {
		return OpenResponse{}, fmt.Errorf("read response: %w", err)
	}

	var resp OpenResponse
	if err := json.Unmarshal(buf[:n], &resp); err != nil {
		return OpenResponse{}, fmt.Errorf("unmarshal response: %w", err)
	}
	return resp, nil
}
