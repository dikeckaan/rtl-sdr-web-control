package fmradio

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/go-chi/chi/v5"
	"github.com/kaandikec/sdr/internal/audio"
	"github.com/kaandikec/sdr/internal/config"
	"github.com/kaandikec/sdr/internal/process"
	"github.com/kaandikec/sdr/internal/services"
)

type Service struct {
	mu       sync.RWMutex
	cfg      config.FMRadioConfig
	sdrCfg   config.SDRConfig
	proc     *process.Manager
	pipeline *process.Pipeline
	streamer *audio.Streamer
	status   services.Status
	freq     string
}

func New(cfg config.FMRadioConfig, sdrCfg config.SDRConfig, proc *process.Manager) *Service {
	return &Service{
		cfg:      cfg,
		sdrCfg:   sdrCfg,
		proc:     proc,
		streamer: audio.NewStreamer(),
		status:   services.StatusStopped,
		freq:     cfg.DefaultFreq,
	}
}

func (s *Service) ID() string { return "fmradio" }

func (s *Service) Info() services.ServiceInfo {
	return services.ServiceInfo{
		ID:          "fmradio",
		Name:        "FM Radyo",
		Description: "FM radyo alıcısı ve yayın akışı",
		Category:    services.CategoryNative,
		Status:      s.Status(),
		Icon:        "radio",
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
	if err := s.startPipeline(); err != nil {
		s.status = services.StatusError
		return err
	}

	s.status = services.StatusRunning
	log.Printf("[fmradio] Started on %s", s.freq)
	return nil
}

func (s *Service) startPipeline() error {
	gain := fmt.Sprintf("%d", s.sdrCfg.Gain)
	sampleRate := fmt.Sprintf("%d", s.cfg.SampleRate)

	pipeline, err := process.NewPipeline("fmradio", []process.PipelineCmd{
		{
			Name: "rtl_fm",
			Args: []string{"-M", "wbfm", "-f", s.freq, "-s", sampleRate, "-g", gain, "-"},
		},
		{
			Name: "ffmpeg",
			Args: []string{
				"-f", "s16le", "-ar", sampleRate, "-ac", "1", "-i", "pipe:0",
				"-af", "aresample=48000",
				"-b:a", s.cfg.AudioBitrate,
				"-f", "mp3", "pipe:1",
			},
		},
	})
	if err != nil {
		return fmt.Errorf("creating pipeline: %w", err)
	}

	if err := pipeline.Start(); err != nil {
		return fmt.Errorf("starting pipeline: %w", err)
	}

	s.pipeline = pipeline

	// Stream audio from the pipeline to WebSocket clients
	go s.streamer.StreamFrom(pipeline.Stdout)

	// Monitor pipeline exit
	go func() {
		<-pipeline.ExitCh
		s.mu.Lock()
		if s.status == services.StatusRunning {
			s.status = services.StatusStopped
		}
		s.mu.Unlock()
	}()

	return nil
}

func (s *Service) Stop(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.pipeline != nil {
		s.status = services.StatusStopping
		s.pipeline.Stop()
		s.pipeline = nil
	}
	s.status = services.StatusStopped
	log.Printf("[fmradio] Stopped")
	return nil
}

func (s *Service) Tune(freq string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.status != services.StatusRunning {
		s.freq = freq
		return nil
	}

	// Stop current pipeline and start new one with new frequency
	if s.pipeline != nil {
		s.pipeline.Stop()
		s.pipeline = nil
	}

	s.freq = freq
	if err := s.startPipeline(); err != nil {
		s.status = services.StatusError
		return err
	}

	log.Printf("[fmradio] Tuned to %s", freq)
	return nil
}

func (s *Service) RegisterRoutes(r chi.Router) {
	r.Route("/api/v1/fm", func(r chi.Router) {
		r.Get("/state", s.handleGetState)
		r.Post("/tune", s.handleTune)
	})
	r.HandleFunc("/ws/audio/fm", s.streamer.HandleWS)
	r.Get("/api/v1/fm/stream", s.streamer.HandleHTTPStream)
}

func (s *Service) handleGetState(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	resp := map[string]any{
		"freq":    s.freq,
		"status":  s.status,
		"presets": s.cfg.Presets,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (s *Service) handleTune(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Freq string `json:"freq"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Freq == "" {
		http.Error(w, `{"error":"freq required"}`, http.StatusBadRequest)
		return
	}

	if err := s.Tune(req.Freq); err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err), http.StatusInternalServerError)
		return
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"freq":   s.freq,
		"status": s.status,
	})
}
