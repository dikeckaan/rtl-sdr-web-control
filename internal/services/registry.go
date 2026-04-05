package services

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/kaandikec/sdr/internal/sdr"
)

type Registry struct {
	mu       sync.RWMutex
	services map[string]Service
	active   string
	device   *sdr.DeviceLock
}

func NewRegistry(device *sdr.DeviceLock) *Registry {
	return &Registry{
		services: make(map[string]Service),
		device:   device,
	}
}

func (r *Registry) Register(s Service) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.services[s.ID()] = s
	log.Printf("[registry] Registered service: %s", s.ID())
}

func (r *Registry) Get(id string) (Service, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	s, ok := r.services[id]
	return s, ok
}

func (r *Registry) All() []ServiceInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	infos := make([]ServiceInfo, 0, len(r.services))
	for _, s := range r.services {
		infos = append(infos, s.Info())
	}
	return infos
}

func (r *Registry) ActiveID() string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.active
}

func (r *Registry) Start(ctx context.Context, id string) error {
	r.mu.Lock()
	svc, ok := r.services[id]
	if !ok {
		r.mu.Unlock()
		return fmt.Errorf("service %s not found", id)
	}

	// Stop current active service if different
	if r.active != "" && r.active != id {
		activeID := r.active
		activeSvc := r.services[activeID]
		r.mu.Unlock()

		log.Printf("[registry] Stopping active service %s before starting %s", activeID, id)
		if err := activeSvc.Stop(ctx); err != nil {
			log.Printf("[registry] Warning: failed to stop %s: %v", activeID, err)
		}
		r.device.Release(activeID)

		r.mu.Lock()
	}

	r.mu.Unlock()

	// Acquire device lock for native services
	if svc.Info().Category == CategoryNative {
		if err := r.device.Acquire(id); err != nil {
			return fmt.Errorf("acquiring device: %w", err)
		}
	}

	if err := svc.Start(ctx); err != nil {
		if svc.Info().Category == CategoryNative {
			r.device.Release(id)
		}
		return fmt.Errorf("starting %s: %w", id, err)
	}

	r.mu.Lock()
	r.active = id
	r.mu.Unlock()

	log.Printf("[registry] Service %s is now active", id)
	return nil
}

func (r *Registry) Stop(ctx context.Context, id string) error {
	r.mu.Lock()
	svc, ok := r.services[id]
	if !ok {
		r.mu.Unlock()
		return fmt.Errorf("service %s not found", id)
	}
	r.mu.Unlock()

	if err := svc.Stop(ctx); err != nil {
		return fmt.Errorf("stopping %s: %w", id, err)
	}

	r.device.Release(id)

	r.mu.Lock()
	if r.active == id {
		r.active = ""
	}
	r.mu.Unlock()

	return nil
}

func (r *Registry) StopAll(ctx context.Context) {
	r.mu.RLock()
	ids := make([]string, 0, len(r.services))
	for id := range r.services {
		ids = append(ids, id)
	}
	r.mu.RUnlock()

	for _, id := range ids {
		r.Stop(ctx, id)
	}
}
