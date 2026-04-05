package dockersvc

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"sync"

	"github.com/go-chi/chi/v5"
	"github.com/kaandikec/sdr/internal/config"
	"github.com/kaandikec/sdr/internal/docker"
	"github.com/kaandikec/sdr/internal/services"
)

// Service wraps a Docker container as a Service.
type Service struct {
	mu         sync.RWMutex
	id         string
	cfg        config.DockerService
	client     *docker.Client
	status     services.Status
	projectDir string
}

func New(id string, cfg config.DockerService, client *docker.Client, baseDir string) *Service {
	projectDir := ""
	if cfg.ComposeFile != "" {
		projectDir = filepath.Join(baseDir, filepath.Dir(cfg.ComposeFile))
	} else {
		projectDir = filepath.Join(baseDir, id)
	}

	return &Service{
		id:         id,
		cfg:        cfg,
		client:     client,
		status:     services.StatusStopped,
		projectDir: projectDir,
	}
}

func (s *Service) ID() string { return s.id }

func (s *Service) Info() services.ServiceInfo {
	return services.ServiceInfo{
		ID:          s.id,
		Name:        s.cfg.Name,
		Description: s.cfg.Description,
		Category:    services.CategoryDocker,
		Status:      s.Status(),
		Icon:        "container",
	}
}

func (s *Service) Status() services.Status {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.status
}

func (s *Service) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.status == services.StatusRunning {
		return nil
	}

	s.status = services.StatusStarting

	var err error
	if s.cfg.ComposeFile != "" || s.cfg.Image == "" {
		err = s.client.ComposeUp(ctx, s.projectDir)
	} else {
		err = s.client.RunContainer(ctx, docker.ContainerOpts{
			Name:       "sdr-" + s.id,
			Image:      s.cfg.Image,
			Ports:      []string{"8090:" + itoa(s.cfg.Port)},
			Devices:    []string{"/dev/bus/usb:/dev/bus/usb"},
			Privileged: true,
			Restart:    "unless-stopped",
			Env:        s.cfg.Env,
		})
	}

	if err != nil {
		s.status = services.StatusError
		return err
	}

	s.status = services.StatusRunning
	log.Printf("[docker:%s] Started", s.id)
	return nil
}

func (s *Service) Stop(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.status = services.StatusStopping

	var err error
	if s.cfg.ComposeFile != "" || s.cfg.Image == "" {
		err = s.client.ComposeDown(ctx, s.projectDir)
	} else {
		err = s.client.StopContainer(ctx, "sdr-"+s.id)
	}

	if err != nil {
		log.Printf("[docker:%s] Stop error: %v", s.id, err)
	}

	s.status = services.StatusStopped
	log.Printf("[docker:%s] Stopped", s.id)
	return nil
}

func (s *Service) RegisterRoutes(r chi.Router) {
	// Docker services serve their own UI on port 8090
}

func (s *Service) RefreshStatus(ctx context.Context) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.cfg.ComposeFile != "" || s.cfg.Image == "" {
		statuses, err := s.client.ComposeStatus(ctx, s.projectDir)
		if err != nil || len(statuses) == 0 {
			s.status = services.StatusStopped
			return
		}
		allRunning := true
		for _, st := range statuses {
			if !st.Running {
				allRunning = false
				break
			}
		}
		if allRunning {
			s.status = services.StatusRunning
		} else {
			s.status = services.StatusStopped
		}
	} else {
		if s.client.IsContainerRunning(ctx, "sdr-"+s.id) {
			s.status = services.StatusRunning
		} else {
			s.status = services.StatusStopped
		}
	}
}

func itoa(n int) string {
	if n == 0 {
		return "8090"
	}
	return fmt.Sprintf("%d", n)
}
