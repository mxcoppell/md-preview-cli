package ipc

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"sync"
	"syscall"
	"time"
)

// Handler is called when a new open request arrives from a CLI client.
type Handler func(req OpenRequest) OpenResponse

// Server listens on a Unix socket for IPC requests from CLI processes.
type Server struct {
	listener net.Listener
	handler  Handler
	done     chan struct{}
	once     sync.Once
}

// NewServer creates a new IPC server that forwards requests to handler.
// It uses a try-listen-first pattern to avoid racing with another host:
//  1. Try net.Listen without removing the socket.
//  2. If "address already in use", dial the socket to check liveness.
//  3. If live → return ErrHostAlreadyRunning; if stale → remove and retry once.
func NewServer(handler Handler) (*Server, error) {
	path := SocketPath()

	ln, err := net.Listen("unix", path)
	if err == nil {
		return &Server{
			listener: ln,
			handler:  handler,
			done:     make(chan struct{}),
		}, nil
	}

	// Check if the error is "address already in use"
	if !isAddrInUse(err) {
		return nil, fmt.Errorf("listen unix %s: %w", path, err)
	}

	// Socket exists — probe whether a live host owns it
	conn, dialErr := net.DialTimeout("unix", path, 500*time.Millisecond)
	if dialErr == nil {
		conn.Close()
		return nil, ErrHostAlreadyRunning
	}

	// Stale socket — remove and retry once
	os.Remove(path)
	ln, err = net.Listen("unix", path)
	if err != nil {
		return nil, fmt.Errorf("listen unix %s (retry): %w", path, err)
	}

	return &Server{
		listener: ln,
		handler:  handler,
		done:     make(chan struct{}),
	}, nil
}

// isAddrInUse reports whether err is a "bind: address already in use" error.
func isAddrInUse(err error) bool {
	var opErr *net.OpError
	if !errors.As(err, &opErr) {
		return false
	}
	var sysErr *os.SyscallError
	if !errors.As(opErr.Err, &sysErr) {
		return false
	}
	return sysErr.Err == syscall.EADDRINUSE
}

// Serve accepts connections until Close is called.
func (s *Server) Serve() {
	defer close(s.done)
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			return // listener closed
		}
		go s.handleConn(conn)
	}
}

func (s *Server) handleConn(conn net.Conn) {
	defer conn.Close()

	scanner := bufio.NewScanner(conn)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024) // 1MB max
	if !scanner.Scan() {
		return
	}

	var req OpenRequest
	if err := json.Unmarshal(scanner.Bytes(), &req); err != nil {
		resp := OpenResponse{Error: "invalid request"}
		data, _ := json.Marshal(resp)
		conn.Write(append(data, '\n'))
		return
	}

	resp := s.handler(req)
	data, _ := json.Marshal(resp)
	conn.Write(append(data, '\n'))
}

// Close stops the server and removes the socket file.
func (s *Server) Close() {
	s.once.Do(func() {
		s.listener.Close()
		os.Remove(SocketPath())
	})
}

// Wait blocks until the server has stopped.
func (s *Server) Wait() {
	<-s.done
}
