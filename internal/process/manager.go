package process

import (
	"context"
	"fmt"
	"io"
	"log"
	"os/exec"
	"sync"
	"syscall"
	"time"
)

type Manager struct {
	mu      sync.Mutex
	running map[string]*ManagedProcess
}

type ManagedProcess struct {
	ID      string
	Cmd     *exec.Cmd
	Cancel  context.CancelFunc
	Stdout  io.ReadCloser
	Stderr  io.ReadCloser
	ExitCh  chan error
	Started time.Time
}

func NewManager() *Manager {
	return &Manager{
		running: make(map[string]*ManagedProcess),
	}
}

func (m *Manager) Start(id string, name string, args ...string) (*ManagedProcess, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if p, exists := m.running[id]; exists {
		if p.Cmd.Process != nil {
			return nil, fmt.Errorf("process %s already running (pid %d)", id, p.Cmd.Process.Pid)
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		cancel()
		return nil, fmt.Errorf("stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		cancel()
		return nil, fmt.Errorf("stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		cancel()
		return nil, fmt.Errorf("starting %s: %w", name, err)
	}

	proc := &ManagedProcess{
		ID:      id,
		Cmd:     cmd,
		Cancel:  cancel,
		Stdout:  stdout,
		Stderr:  stderr,
		ExitCh:  make(chan error, 1),
		Started: time.Now(),
	}

	go func() {
		err := cmd.Wait()
		proc.ExitCh <- err
		m.mu.Lock()
		delete(m.running, id)
		m.mu.Unlock()
		log.Printf("[process] %s exited: %v", id, err)
	}()

	m.running[id] = proc
	log.Printf("[process] Started %s (pid %d): %s %v", id, cmd.Process.Pid, name, args)
	return proc, nil
}

func (m *Manager) Stop(id string) error {
	m.mu.Lock()
	proc, exists := m.running[id]
	m.mu.Unlock()

	if !exists {
		return nil
	}

	if proc.Cmd.Process == nil {
		return nil
	}

	// Send SIGTERM to the process group
	pgid, err := syscall.Getpgid(proc.Cmd.Process.Pid)
	if err == nil {
		syscall.Kill(-pgid, syscall.SIGTERM)
	} else {
		proc.Cmd.Process.Signal(syscall.SIGTERM)
	}

	// Wait up to 5 seconds for graceful shutdown
	select {
	case <-proc.ExitCh:
		return nil
	case <-time.After(5 * time.Second):
		// Force kill
		if pgid > 0 {
			syscall.Kill(-pgid, syscall.SIGKILL)
		} else {
			proc.Cmd.Process.Kill()
		}
		proc.Cancel()
		<-proc.ExitCh
		return nil
	}
}

func (m *Manager) IsRunning(id string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	_, exists := m.running[id]
	return exists
}

func (m *Manager) StopAll() {
	m.mu.Lock()
	ids := make([]string, 0, len(m.running))
	for id := range m.running {
		ids = append(ids, id)
	}
	m.mu.Unlock()

	for _, id := range ids {
		m.Stop(id)
	}
}

func (m *Manager) Get(id string) *ManagedProcess {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.running[id]
}
