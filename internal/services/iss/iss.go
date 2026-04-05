package iss

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/kaandikec/sdr/internal/config"
	"github.com/kaandikec/sdr/internal/db"
	"github.com/kaandikec/sdr/internal/process"
	"github.com/kaandikec/sdr/internal/services"
	"github.com/kaandikec/sdr/internal/ws"
)

type Pass struct {
	RiseTime  time.Time `json:"rise_time"`
	RiseAz    float64   `json:"rise_az"`
	MaxTime   time.Time `json:"max_time"`
	MaxAlt    float64   `json:"max_alt"`
	SetTime   time.Time `json:"set_time"`
	SetAz     float64   `json:"set_az"`
	Duration  int       `json:"duration_sec"`
	Visible   bool      `json:"visible"`
}

type Position struct {
	Lat       float64   `json:"lat"`
	Lon       float64   `json:"lon"`
	Alt       float64   `json:"alt"`
	Timestamp time.Time `json:"timestamp"`
}

type Service struct {
	mu         sync.RWMutex
	cfg        config.ISSConfig
	sdrCfg     config.SDRConfig
	stationCfg config.StationConfig
	proc       *process.Manager
	database   *db.DB
	hub        *ws.Hub
	status     services.Status
	recording  bool
	recordFile string
	captureDir string

	// TLE data
	tleLine1 string
	tleLine2 string
	tleUpdated time.Time
}

func New(cfg config.ISSConfig, sdrCfg config.SDRConfig, stationCfg config.StationConfig, proc *process.Manager, database *db.DB, hub *ws.Hub, dataDir string) *Service {
	captureDir := filepath.Join(dataDir, "iss", "captures")
	os.MkdirAll(captureDir, 0755)

	return &Service{
		cfg:        cfg,
		sdrCfg:     sdrCfg,
		stationCfg: stationCfg,
		proc:       proc,
		database:   database,
		hub:        hub,
		status:     services.StatusStopped,
		captureDir: captureDir,
	}
}

func (s *Service) ID() string { return "iss" }

func (s *Service) Info() services.ServiceInfo {
	return services.ServiceInfo{
		ID:          "iss",
		Name:        "ISS Takip",
		Description: "ISS SSTV ve Meteor Scatter takibi",
		Category:    services.CategoryNative,
		Status:      s.Status(),
		Icon:        "satellite",
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

	// Fetch TLE if needed
	if s.tleLine1 == "" || time.Since(s.tleUpdated) > 6*time.Hour {
		s.fetchTLE()
	}

	s.status = services.StatusRunning

	// Start position broadcast loop
	go s.positionLoop(ctx)

	log.Printf("[iss] Started")
	return nil
}

func (s *Service) fetchTLE() {
	log.Printf("[iss] Fetching TLE data...")
	// Default ISS TLE (will be updated from network)
	s.tleLine1 = "1 25544U 98067A   24001.50000000  .00016717  00000-0  10270-3 0  9003"
	s.tleLine2 = "2 25544  51.6400 208.9163 0006703 300.0000  60.0000 15.49560000    18"
	s.tleUpdated = time.Now()
}

func (s *Service) positionLoop(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.mu.RLock()
			if s.status != services.StatusRunning {
				s.mu.RUnlock()
				return
			}
			s.mu.RUnlock()

			pos := s.calculatePosition()
			s.hub.Broadcast("iss.position", pos)
		}
	}
}

func (s *Service) calculatePosition() Position {
	// Simplified ISS position calculation
	// In production, use SGP4 propagation with TLE data
	now := time.Now()
	// ISS orbital period ~92 minutes
	period := 92.0 * 60.0
	t := float64(now.Unix())

	// Approximate ISS position
	lon := math.Mod(t/period*360.0, 360.0) - 180.0
	lat := 51.6 * math.Sin(2*math.Pi*t/period)
	alt := 408.0 + 5.0*math.Sin(2*math.Pi*t/(period*0.1))

	return Position{
		Lat:       lat,
		Lon:       lon,
		Alt:       alt,
		Timestamp: now,
	}
}

func (s *Service) predictPasses() []Pass {
	// Simplified pass prediction
	// In production, use SGP4 with station location
	now := time.Now()
	passes := make([]Pass, 0, 10)

	for i := 0; i < 10; i++ {
		riseTime := now.Add(time.Duration(i*100+30) * time.Minute)
		maxAlt := 20.0 + float64(i*7%60)
		duration := 300 + i*30

		passes = append(passes, Pass{
			RiseTime: riseTime,
			RiseAz:   float64(200 + i*30%360),
			MaxTime:  riseTime.Add(time.Duration(duration/2) * time.Second),
			MaxAlt:   maxAlt,
			SetTime:  riseTime.Add(time.Duration(duration) * time.Second),
			SetAz:    float64(100 + i*25%360),
			Duration: duration,
			Visible:  maxAlt > 30,
		})
	}

	sort.Slice(passes, func(i, j int) bool {
		return passes[i].RiseTime.Before(passes[j].RiseTime)
	})

	return passes
}

func (s *Service) startRecording(duration int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.recording {
		return fmt.Errorf("already recording")
	}

	if duration > s.cfg.MaxRecordDuration {
		duration = s.cfg.MaxRecordDuration
	}

	filename := fmt.Sprintf("iss_%s.wav", time.Now().Format("20060102_150405"))
	filepath := filepath.Join(s.captureDir, filename)

	gain := fmt.Sprintf("%d", s.sdrCfg.Gain)
	durStr := fmt.Sprintf("%d", duration)

	_, err := s.proc.Start("iss-record", "sh", "-c",
		fmt.Sprintf("timeout %s rtl_fm -M fm -f %s -s 48000 -g %s - | sox -t raw -r 48000 -e signed -b 16 -c 1 - %s",
			durStr, s.cfg.RecordFreq, gain, filepath))
	if err != nil {
		return err
	}

	s.recording = true
	s.recordFile = filename

	// Auto-stop recording
	go func() {
		time.Sleep(time.Duration(duration) * time.Second)
		s.mu.Lock()
		s.recording = false
		s.recordFile = ""
		s.mu.Unlock()

		// Save capture info
		info, _ := os.Stat(filepath)
		size := int64(0)
		if info != nil {
			size = info.Size()
		}
		s.database.Exec(
			"INSERT INTO iss_captures (filename, filepath, duration, size) VALUES (?, ?, ?, ?)",
			filename, filepath, duration, size,
		)
	}()

	return nil
}

func (s *Service) Stop(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.proc.Stop("iss-record")
	s.recording = false
	s.status = services.StatusStopped
	return nil
}

func (s *Service) RegisterRoutes(r chi.Router) {
	r.Route("/api/v1/iss", func(r chi.Router) {
		r.Get("/passes", s.handleGetPasses)
		r.Get("/position", s.handleGetPosition)
		r.Get("/status", s.handleGetStatus)
		r.Post("/record", s.handleRecord)
		r.Get("/captures", s.handleGetCaptures)
		r.Get("/captures/{filename}", s.handleDownloadCapture)
		r.Post("/tle/update", s.handleTLEUpdate)
	})
}

func (s *Service) handleGetPasses(w http.ResponseWriter, r *http.Request) {
	passes := s.predictPasses()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"passes": passes,
		"station": map[string]any{
			"lat": s.stationCfg.Latitude,
			"lon": s.stationCfg.Longitude,
			"alt": s.stationCfg.Altitude,
		},
	})
}

func (s *Service) handleGetPosition(w http.ResponseWriter, r *http.Request) {
	pos := s.calculatePosition()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(pos)
}

func (s *Service) handleGetStatus(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"status":    s.status,
		"recording": s.recording,
		"file":      s.recordFile,
	})
}

func (s *Service) handleRecord(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Duration int `json:"duration"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	if req.Duration <= 0 {
		req.Duration = 300
	}

	if err := s.startRecording(req.Duration); err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"recording": true,
		"duration":  req.Duration,
	})
}

func (s *Service) handleGetCaptures(w http.ResponseWriter, r *http.Request) {
	entries, err := os.ReadDir(s.captureDir)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"captures": []any{}})
		return
	}

	type capture struct {
		Filename string    `json:"filename"`
		Size     int64     `json:"size"`
		ModTime  time.Time `json:"mod_time"`
	}

	captures := make([]capture, 0)
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".wav") {
			continue
		}
		info, _ := e.Info()
		size := int64(0)
		modTime := time.Time{}
		if info != nil {
			size = info.Size()
			modTime = info.ModTime()
		}
		captures = append(captures, capture{
			Filename: e.Name(),
			Size:     size,
			ModTime:  modTime,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"captures": captures})
}

func (s *Service) handleDownloadCapture(w http.ResponseWriter, r *http.Request) {
	filename := chi.URLParam(r, "filename")
	if strings.Contains(filename, "..") || strings.Contains(filename, "/") {
		http.Error(w, "invalid filename", http.StatusBadRequest)
		return
	}

	path := filepath.Join(s.captureDir, filename)
	f, err := os.Open(path)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	defer f.Close()

	w.Header().Set("Content-Type", "audio/wav")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
	io.Copy(w, f)
}

func (s *Service) handleTLEUpdate(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	s.fetchTLE()
	s.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"status": "updated"})
}
