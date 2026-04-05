package services

import (
	"context"

	"github.com/go-chi/chi/v5"
)

type Category string

const (
	CategoryNative Category = "native"
	CategoryDocker Category = "docker"
)

type Status string

const (
	StatusStopped  Status = "stopped"
	StatusStarting Status = "starting"
	StatusRunning  Status = "running"
	StatusStopping Status = "stopping"
	StatusError    Status = "error"
)

type ServiceInfo struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Category    Category `json:"category"`
	Status      Status   `json:"status"`
	Icon        string   `json:"icon"`
}

type Service interface {
	ID() string
	Info() ServiceInfo
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	Status() Status
	RegisterRoutes(r chi.Router)
}
