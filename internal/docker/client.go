package docker

import (
	"context"
	"fmt"
	"log"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// Client manages Docker containers for pre-built SDR services.
// Uses docker CLI for simplicity and reliability on Raspberry Pi.
type Client struct {
	mu sync.Mutex
}

func NewClient() *Client {
	return &Client{}
}

type ContainerStatus struct {
	Name    string `json:"name"`
	State   string `json:"state"`
	Status  string `json:"status"`
	Running bool   `json:"running"`
}

// ComposeUp starts a docker-compose project.
func (c *Client) ComposeUp(ctx context.Context, projectDir string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	cmd := exec.CommandContext(ctx, "docker", "compose", "up", "-d")
	cmd.Dir = projectDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("compose up: %s: %w", string(output), err)
	}
	log.Printf("[docker] compose up in %s", projectDir)
	return nil
}

// ComposeDown stops a docker-compose project.
func (c *Client) ComposeDown(ctx context.Context, projectDir string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	cmd := exec.CommandContext(ctx, "docker", "compose", "down")
	cmd.Dir = projectDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("compose down: %s: %w", string(output), err)
	}
	log.Printf("[docker] compose down in %s", projectDir)
	return nil
}

// ComposeStatus checks if containers in a compose project are running.
func (c *Client) ComposeStatus(ctx context.Context, projectDir string) ([]ContainerStatus, error) {
	cmd := exec.CommandContext(ctx, "docker", "compose", "ps", "--format", "{{.Name}}|{{.State}}|{{.Status}}")
	cmd.Dir = projectDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, nil // Not running
	}

	var statuses []ContainerStatus
	for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "|", 3)
		if len(parts) < 2 {
			continue
		}
		status := ""
		if len(parts) > 2 {
			status = parts[2]
		}
		statuses = append(statuses, ContainerStatus{
			Name:    parts[0],
			State:   parts[1],
			Status:  status,
			Running: parts[1] == "running",
		})
	}
	return statuses, nil
}

// RunContainer starts a single container with the given configuration.
func (c *Client) RunContainer(ctx context.Context, opts ContainerOpts) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Remove existing container if any
	exec.CommandContext(ctx, "docker", "rm", "-f", opts.Name).Run()

	args := []string{"run", "-d", "--name", opts.Name}

	for _, p := range opts.Ports {
		args = append(args, "-p", p)
	}
	for _, v := range opts.Volumes {
		args = append(args, "-v", v)
	}
	for _, d := range opts.Devices {
		args = append(args, "--device", d)
	}
	for k, v := range opts.Env {
		args = append(args, "-e", fmt.Sprintf("%s=%s", k, v))
	}
	if opts.Privileged {
		args = append(args, "--privileged")
	}
	if opts.Restart != "" {
		args = append(args, "--restart", opts.Restart)
	}

	args = append(args, opts.Image)

	cmd := exec.CommandContext(ctx, "docker", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("docker run: %s: %w", string(output), err)
	}
	log.Printf("[docker] Started container %s from %s", opts.Name, opts.Image)
	return nil
}

// StopContainer stops and removes a container.
func (c *Client) StopContainer(ctx context.Context, name string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	stopCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(stopCtx, "docker", "stop", name)
	cmd.Run()

	cmd = exec.CommandContext(ctx, "docker", "rm", "-f", name)
	cmd.Run()

	log.Printf("[docker] Stopped container %s", name)
	return nil
}

// IsContainerRunning checks if a named container is running.
func (c *Client) IsContainerRunning(ctx context.Context, name string) bool {
	cmd := exec.CommandContext(ctx, "docker", "inspect", "-f", "{{.State.Running}}", name)
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(output)) == "true"
}

type ContainerOpts struct {
	Name       string
	Image      string
	Ports      []string
	Volumes    []string
	Devices    []string
	Env        map[string]string
	Privileged bool
	Restart    string
}
