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

// Pipeline chains multiple processes with pipes.
// Example: rtl_fm | ffmpeg
type Pipeline struct {
	ID       string
	Commands []*exec.Cmd
	Cancel   context.CancelFunc
	Stdout   io.ReadCloser // stdout of the last process
	ctx      context.Context
	mu       sync.Mutex
	running  bool
	ExitCh   chan error
}

type PipelineCmd struct {
	Name string
	Args []string
}

func NewPipeline(id string, cmds []PipelineCmd) (*Pipeline, error) {
	if len(cmds) == 0 {
		return nil, fmt.Errorf("pipeline requires at least one command")
	}

	ctx, cancel := context.WithCancel(context.Background())
	p := &Pipeline{
		ID:     id,
		Cancel: cancel,
		ctx:    ctx,
		ExitCh: make(chan error, 1),
	}

	for _, c := range cmds {
		cmd := exec.CommandContext(ctx, c.Name, c.Args...)
		cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
		p.Commands = append(p.Commands, cmd)
	}

	// Chain stdin/stdout
	for i := 0; i < len(p.Commands)-1; i++ {
		pipe, err := p.Commands[i].StdoutPipe()
		if err != nil {
			cancel()
			return nil, fmt.Errorf("pipe %d: %w", i, err)
		}
		p.Commands[i+1].Stdin = pipe
	}

	// Capture the final stdout
	lastStdout, err := p.Commands[len(p.Commands)-1].StdoutPipe()
	if err != nil {
		cancel()
		return nil, fmt.Errorf("final stdout pipe: %w", err)
	}
	p.Stdout = lastStdout

	return p, nil
}

func (p *Pipeline) Start() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Start all commands in order
	for i, cmd := range p.Commands {
		if err := cmd.Start(); err != nil {
			// Kill already started processes
			for j := 0; j < i; j++ {
				p.Commands[j].Process.Kill()
			}
			p.Cancel()
			return fmt.Errorf("starting command %d (%s): %w", i, cmd.Path, err)
		}
		log.Printf("[pipeline:%s] Started %s (pid %d)", p.ID, cmd.Path, cmd.Process.Pid)
	}

	p.running = true

	// Wait for all processes in background
	go func() {
		var firstErr error
		for _, cmd := range p.Commands {
			if err := cmd.Wait(); err != nil && firstErr == nil {
				firstErr = err
			}
		}
		p.mu.Lock()
		p.running = false
		p.mu.Unlock()
		p.ExitCh <- firstErr
		log.Printf("[pipeline:%s] All processes exited: %v", p.ID, firstErr)
	}()

	return nil
}

func (p *Pipeline) Stop() error {
	p.mu.Lock()
	if !p.running {
		p.mu.Unlock()
		return nil
	}
	p.mu.Unlock()

	// Send SIGTERM to all process groups
	for _, cmd := range p.Commands {
		if cmd.Process != nil {
			pgid, err := syscall.Getpgid(cmd.Process.Pid)
			if err == nil {
				syscall.Kill(-pgid, syscall.SIGTERM)
			}
		}
	}

	select {
	case <-p.ExitCh:
		return nil
	case <-time.After(5 * time.Second):
		for _, cmd := range p.Commands {
			if cmd.Process != nil {
				cmd.Process.Kill()
			}
		}
		p.Cancel()
		<-p.ExitCh
		return nil
	}
}

func (p *Pipeline) IsRunning() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.running
}
