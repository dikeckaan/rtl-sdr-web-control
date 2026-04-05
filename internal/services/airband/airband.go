package airband

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

type channelStream struct {
	Channel  config.AirbandChannel
	Streamer *audio.Streamer
}

type Service struct {
	mu       sync.RWMutex
	cfg      config.AirbandConfig
	sdrCfg   config.SDRConfig
	proc     *process.Manager
	channels []channelStream
	status   services.Status
}

func New(cfg config.AirbandConfig, sdrCfg config.SDRConfig, proc *process.Manager) *Service {
	channels := make([]channelStream, len(cfg.Channels))
	for i, ch := range cfg.Channels {
		channels[i] = channelStream{
			Channel:  ch,
			Streamer: audio.NewStreamer(),
		}
	}
	return &Service{
		cfg:      cfg,
		sdrCfg:   sdrCfg,
		proc:     proc,
		channels: channels,
		status:   services.StatusStopped,
	}
}

func (s *Service) ID() string { return "airband" }

func (s *Service) Info() services.ServiceInfo {
	return services.ServiceInfo{
		ID:          "airband",
		Name:        "Havacılık Bandı",
		Description: "Havacılık frekansları dinleme",
		Category:    services.CategoryNative,
		Status:      s.Status(),
		Icon:        "plane",
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

	// Start a separate rtl_fm | ffmpeg pipeline for each channel
	for i, ch := range s.channels {
		freq := fmt.Sprintf("%.3fM", ch.Channel.Freq)
		gain := fmt.Sprintf("%d", s.cfg.Gain)
		procID := fmt.Sprintf("airband-%d", i)

		p, err := s.proc.Start(procID, "sh", "-c",
			fmt.Sprintf("rtl_fm -M am -f %s -s 12000 -g %s - | ffmpeg -f s16le -ar 12000 -ac 1 -i pipe:0 -af 'highpass=f=300,lowpass=f=3000,volume=3' -ar 48000 -b:a 64k -f mp3 pipe:1", freq, gain))
		if err != nil {
			log.Printf("[airband] Failed to start channel %s: %v", ch.Channel.Name, err)
			continue
		}

		go s.channels[i].Streamer.StreamFrom(p.Stdout)
	}

	s.status = services.StatusRunning
	log.Printf("[airband] Started %d channels", len(s.channels))
	return nil
}

func (s *Service) Stop(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i := range s.channels {
		procID := fmt.Sprintf("airband-%d", i)
		s.proc.Stop(procID)
	}

	s.status = services.StatusStopped
	return nil
}

func (s *Service) RegisterRoutes(r chi.Router) {
	r.Get("/api/v1/airband/channels", s.handleGetChannels)
	for i := range s.channels {
		idx := i
		path := fmt.Sprintf("/ws/audio/airband/%d", idx)
		r.HandleFunc(path, s.channels[idx].Streamer.HandleWS)
		streamPath := fmt.Sprintf("/api/v1/airband/stream/%d", idx)
		r.Get(streamPath, s.channels[idx].Streamer.HandleHTTPStream)
	}
}

func (s *Service) handleGetChannels(w http.ResponseWriter, r *http.Request) {
	type channelInfo struct {
		Index      int     `json:"index"`
		Name       string  `json:"name"`
		Freq       float64 `json:"freq"`
		Modulation string  `json:"modulation"`
	}
	channels := make([]channelInfo, len(s.channels))
	for i, ch := range s.channels {
		channels[i] = channelInfo{
			Index:      i,
			Name:       ch.Channel.Name,
			Freq:       ch.Channel.Freq,
			Modulation: ch.Channel.Modulation,
		}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"channels": channels,
		"status":   s.Status(),
	})
}
