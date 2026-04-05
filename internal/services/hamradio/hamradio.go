package hamradio

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
	cfg      config.HamRadioConfig
	sdrCfg   config.SDRConfig
	proc     *process.Manager
	pipeline *process.Pipeline
	streamer *audio.Streamer
	status   services.Status
	freq     string
	mode     string
}

func New(cfg config.HamRadioConfig, sdrCfg config.SDRConfig, proc *process.Manager) *Service {
	return &Service{
		cfg:      cfg,
		sdrCfg:   sdrCfg,
		proc:     proc,
		streamer: audio.NewStreamer(),
		status:   services.StatusStopped,
		freq:     cfg.DefaultFreq,
		mode:     cfg.DefaultMode,
	}
}

func (s *Service) ID() string { return "hamradio" }

func (s *Service) Info() services.ServiceInfo {
	return services.ServiceInfo{
		ID:          "hamradio",
		Name:        "Amatör Radyo",
		Description: "VHF/UHF amatör radyo alıcısı",
		Category:    services.CategoryNative,
		Status:      s.Status(),
		Icon:        "antenna",
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
	log.Printf("[hamradio] Started on %s mode=%s", s.freq, s.mode)
	return nil
}

func (s *Service) startPipeline() error {
	gain := fmt.Sprintf("%d", s.sdrCfg.Gain)

	// Mode-dependent settings
	rtlMode := s.mode
	sampleRate := "24000"
	if s.mode == "usb" || s.mode == "lsb" {
		sampleRate = "12000"
	}
	if s.mode == "am" {
		sampleRate = "12000"
	}

	pipeline, err := process.NewPipeline("hamradio", []process.PipelineCmd{
		{
			Name: "rtl_fm",
			Args: []string{"-M", rtlMode, "-f", s.freq, "-s", sampleRate, "-g", gain, "-"},
		},
		{
			Name: "ffmpeg",
			Args: []string{
				"-f", "s16le", "-ar", sampleRate, "-ac", "1", "-i", "pipe:0",
				"-af", "highpass=f=200,lowpass=f=5000,volume=2",
				"-ar", "48000", "-b:a", "96k",
				"-f", "mp3", "pipe:1",
			},
		},
	})
	if err != nil {
		return err
	}

	if err := pipeline.Start(); err != nil {
		return err
	}

	s.pipeline = pipeline
	go s.streamer.StreamFrom(pipeline.Stdout)
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
	return nil
}

func (s *Service) Tune(freq, mode string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if mode != "" {
		s.mode = mode
	}
	if freq != "" {
		s.freq = freq
	}

	if s.status != services.StatusRunning {
		return nil
	}

	if s.pipeline != nil {
		s.pipeline.Stop()
		s.pipeline = nil
	}

	if err := s.startPipeline(); err != nil {
		s.status = services.StatusError
		return err
	}
	log.Printf("[hamradio] Tuned to %s mode=%s", s.freq, s.mode)
	return nil
}

func (s *Service) RegisterRoutes(r chi.Router) {
	r.Route("/api/v1/ham", func(r chi.Router) {
		r.Get("/state", s.handleGetState)
		r.Post("/tune", s.handleTune)
	})
	r.HandleFunc("/ws/audio/ham", s.streamer.HandleWS)
	r.Get("/api/v1/ham/stream", s.streamer.HandleHTTPStream)
}

func (s *Service) handleGetState(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"freq":    s.freq,
		"mode":    s.mode,
		"status":  s.status,
		"presets": s.cfg.Presets,
		"modes":   []string{"fm", "am", "usb", "lsb"},
	})
}

func (s *Service) handleTune(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Freq string `json:"freq"`
		Mode string `json:"mode"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request"}`, http.StatusBadRequest)
		return
	}
	if err := s.Tune(req.Freq, req.Mode); err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err), http.StatusInternalServerError)
		return
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"freq": s.freq, "mode": s.mode, "status": s.status})
}
