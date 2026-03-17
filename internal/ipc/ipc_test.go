package ipc

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSocketPath(t *testing.T) {
	path := SocketPath()
	if path == "" {
		t.Fatal("SocketPath returned empty string")
	}
	expected := fmt.Sprintf("md-preview-cli-%d.sock", os.Getuid())
	if filepath.Base(path) != expected {
		t.Errorf("unexpected socket filename: got %s, want %s", filepath.Base(path), expected)
	}
}

func TestIsHostRunning_NoHost(t *testing.T) {
	// Clean up any existing socket
	os.Remove(SocketPath())
	if IsHostRunning() {
		t.Error("IsHostRunning should return false when no host is listening")
	}
}

func TestCleanStaleSocket_NoSocket(t *testing.T) {
	os.Remove(SocketPath())
	// Should not panic
	CleanStaleSocket()
}

func TestCleanStaleSocket_StaleSocket(t *testing.T) {
	path := SocketPath()
	os.Remove(path)
	// Create a stale socket file (just a regular file)
	os.WriteFile(path, []byte("stale"), 0600)
	defer os.Remove(path)

	CleanStaleSocket()

	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("CleanStaleSocket should have removed the stale socket")
	}
}

func TestServerAndDial(t *testing.T) {
	os.Remove(SocketPath())

	srv, err := NewServer(func(req OpenRequest) OpenResponse {
		return OpenResponse{OK: true, WindowID: "w-1"}
	})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	defer srv.Close()

	go srv.Serve()

	// Verify host is running
	if !IsHostRunning() {
		t.Fatal("IsHostRunning should return true")
	}

	// Send a request
	conn, err := Dial()
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	defer conn.Close()

	resp, err := SendOpen(conn, "/tmp/test-config.json")
	if err != nil {
		t.Fatalf("SendOpen: %v", err)
	}
	if !resp.OK {
		t.Errorf("expected OK=true, got OK=false, error=%s", resp.Error)
	}
	if resp.WindowID != "w-1" {
		t.Errorf("expected WindowID=w-1, got %s", resp.WindowID)
	}
}

func TestServerInvalidJSON(t *testing.T) {
	os.Remove(SocketPath())

	srv, err := NewServer(func(req OpenRequest) OpenResponse {
		return OpenResponse{OK: true}
	})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	defer srv.Close()

	go srv.Serve()

	conn, err := Dial()
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	defer conn.Close()

	// Send invalid JSON
	conn.Write([]byte("not-json\n"))
	buf := make([]byte, 4096)
	n, err := conn.Read(buf)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}

	var resp OpenResponse
	if err := json.Unmarshal(buf[:n], &resp); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if resp.Error == "" {
		t.Error("expected error response for invalid JSON")
	}
}

func TestNewServer_HostAlreadyRunning(t *testing.T) {
	os.Remove(SocketPath())

	srv1, err := NewServer(func(req OpenRequest) OpenResponse {
		return OpenResponse{OK: true}
	})
	if err != nil {
		t.Fatalf("first NewServer: %v", err)
	}
	defer srv1.Close()
	go srv1.Serve()

	_, err = NewServer(func(req OpenRequest) OpenResponse {
		return OpenResponse{OK: true}
	})
	if !errors.Is(err, ErrHostAlreadyRunning) {
		t.Errorf("expected ErrHostAlreadyRunning, got: %v", err)
	}
}

func TestNewServer_StaleSocket(t *testing.T) {
	path := SocketPath()
	os.Remove(path)
	os.WriteFile(path, []byte("stale"), 0600)

	srv, err := NewServer(func(req OpenRequest) OpenResponse {
		return OpenResponse{OK: true}
	})
	if err != nil {
		t.Fatalf("NewServer with stale socket: %v", err)
	}
	defer srv.Close()
	// Should have cleaned up stale socket and started successfully
}

func TestOpenResponse_Reused(t *testing.T) {
	resp := OpenResponse{OK: true, WindowID: "w-1", Reused: true}
	data, _ := json.Marshal(resp)
	s := string(data)
	if !strings.Contains(s, `"reused":true`) {
		t.Errorf("expected reused field in JSON: %s", s)
	}

	resp2 := OpenResponse{OK: true, WindowID: "w-2"}
	data2, _ := json.Marshal(resp2)
	if strings.Contains(string(data2), "reused") {
		t.Errorf("reused should be omitted when false: %s", string(data2))
	}
}

func TestServerAndDial_Reused(t *testing.T) {
	os.Remove(SocketPath())

	srv, err := NewServer(func(req OpenRequest) OpenResponse {
		return OpenResponse{OK: true, WindowID: "w-1", Reused: true}
	})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	defer srv.Close()
	go srv.Serve()

	conn, err := Dial()
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	defer conn.Close()

	resp, err := SendOpen(conn, "/tmp/test.json")
	if err != nil {
		t.Fatalf("SendOpen: %v", err)
	}
	if !resp.Reused {
		t.Error("expected Reused=true")
	}
}
