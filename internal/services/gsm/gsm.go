package gsm

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/kaandikec/sdr/internal/config"
	"github.com/kaandikec/sdr/internal/db"
	"github.com/kaandikec/sdr/internal/process"
	"github.com/kaandikec/sdr/internal/services"
)

type CellResult struct {
	Freq     float64 `json:"freq_mhz"`
	ARFCN   int     `json:"arfcn"`
	Band     string  `json:"band"`
	Power    float64 `json:"power_db"`
	Operator string  `json:"operator,omitempty"`
}

type Service struct {
	mu       sync.RWMutex
	cfg      config.GSMConfig
	sdrCfg   config.SDRConfig
	proc     *process.Manager
	database *db.DB
	status   services.Status
	scanning bool
	results  []CellResult
	lastScan time.Time
}

func New(cfg config.GSMConfig, sdrCfg config.SDRConfig, proc *process.Manager, database *db.DB) *Service {
	return &Service{
		cfg:      cfg,
		sdrCfg:   sdrCfg,
		proc:     proc,
		database: database,
		status:   services.StatusStopped,
		results:  make([]CellResult, 0),
	}
}

func (s *Service) ID() string { return "gsm" }

func (s *Service) Info() services.ServiceInfo {
	return services.ServiceInfo{
		ID:          "gsm",
		Name:        "GSM Tarayıcı",
		Description: "GSM baz istasyonu tarama",
		Category:    services.CategoryNative,
		Status:      s.Status(),
		Icon:        "signal",
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
	s.status = services.StatusRunning
	log.Printf("[gsm] Service started (ready to scan)")
	return nil
}

func (s *Service) Stop(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.proc.Stop("gsm-scan")
	s.scanning = false
	s.status = services.StatusStopped
	return nil
}

func (s *Service) startScan(band string) error {
	s.mu.Lock()
	if s.scanning {
		s.mu.Unlock()
		return fmt.Errorf("scan already in progress")
	}
	s.scanning = true
	s.mu.Unlock()

	go func() {
		defer func() {
			s.mu.Lock()
			s.scanning = false
			s.mu.Unlock()
		}()

		var results []CellResult

		if band == "ALL" || band == "GSM900" {
			r := s.scanBand("GSM900", 935e6, 960e6, s.cfg.ThresholdGSM900)
			results = append(results, r...)
		}
		if band == "ALL" || band == "DCS1800" {
			r := s.scanBand("DCS1800", 1805e6, 1880e6, s.cfg.ThresholdDCS1800)
			results = append(results, r...)
		}

		// Resolve operators
		for i := range results {
			results[i].Operator = s.resolveOperator(results[i].ARFCN)
		}

		s.mu.Lock()
		s.results = results
		s.lastScan = time.Now()
		s.mu.Unlock()

		// Save to DB
		resultJSON, _ := json.Marshal(results)
		s.database.Exec(
			"INSERT INTO gsm_scans (band, timestamp, result) VALUES (?, ?, ?)",
			band, time.Now(), string(resultJSON),
		)

		log.Printf("[gsm] Scan complete: %d cells found", len(results))
	}()

	return nil
}

func (s *Service) scanBand(bandName string, startFreq, endFreq, threshold float64) []CellResult {
	gain := fmt.Sprintf("%d", s.cfg.Gain)
	startStr := fmt.Sprintf("%.0f", startFreq)
	endStr := fmt.Sprintf("%.0f", endFreq)

	p, err := s.proc.Start("gsm-scan", "rtl_power",
		"-f", fmt.Sprintf("%s:%s:200k", startStr, endStr),
		"-g", gain, "-i", "10", "-1")
	if err != nil {
		log.Printf("[gsm] Scan error: %v", err)
		return nil
	}

	var results []CellResult
	scanner := bufio.NewScanner(p.Stdout)
	for scanner.Scan() {
		line := scanner.Text()
		cells := s.parseRtlPowerLine(line, bandName, threshold)
		results = append(results, cells...)
	}

	s.proc.Stop("gsm-scan")
	return results
}

func (s *Service) parseRtlPowerLine(line, band string, threshold float64) []CellResult {
	// rtl_power output: date, time, freq_low, freq_high, step, samples, db_values...
	parts := strings.Split(line, ", ")
	if len(parts) < 7 {
		return nil
	}

	freqLow, err := strconv.ParseFloat(strings.TrimSpace(parts[2]), 64)
	if err != nil {
		return nil
	}
	step, err := strconv.ParseFloat(strings.TrimSpace(parts[4]), 64)
	if err != nil {
		return nil
	}

	var results []CellResult
	for i := 6; i < len(parts); i++ {
		power, err := strconv.ParseFloat(strings.TrimSpace(parts[i]), 64)
		if err != nil {
			continue
		}
		if power < threshold {
			continue
		}

		freq := (freqLow + float64(i-6)*step) / 1e6
		arfcn := freqToARFCN(freq, band)

		results = append(results, CellResult{
			Freq:  math.Round(freq*1000) / 1000,
			ARFCN: arfcn,
			Band:  band,
			Power: math.Round(power*10) / 10,
		})
	}

	return results
}

func freqToARFCN(freqMHz float64, band string) int {
	switch band {
	case "GSM900":
		return int(math.Round((freqMHz - 935.0) / 0.2))
	case "DCS1800":
		return int(math.Round((freqMHz-1805.0)/0.2)) + 512
	}
	return 0
}

func (s *Service) resolveOperator(arfcn int) string {
	// Turkish operator ARFCN ranges (approximate)
	for _, op := range s.cfg.Operators {
		switch op.MCCMNC {
		case "286-01": // Turkcell
			if (arfcn >= 1 && arfcn <= 40) || (arfcn >= 512 && arfcn <= 636) {
				return op.Name
			}
		case "286-02": // Vodafone
			if (arfcn >= 41 && arfcn <= 80) || (arfcn >= 637 && arfcn <= 761) {
				return op.Name
			}
		case "286-03": // Turk Telekom
			if (arfcn >= 81 && arfcn <= 124) || (arfcn >= 762 && arfcn <= 885) {
				return op.Name
			}
		}
	}
	return ""
}

func (s *Service) RegisterRoutes(r chi.Router) {
	r.Route("/api/v1/gsm", func(r chi.Router) {
		r.Post("/scan", s.handleScan)
		r.Get("/results", s.handleGetResults)
		r.Get("/status", s.handleGetStatus)
	})
}

func (s *Service) handleScan(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Band string `json:"band"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	if req.Band == "" {
		req.Band = "ALL"
	}

	if err := s.startScan(req.Band); err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"started": true, "band": req.Band})
}

func (s *Service) handleGetResults(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"results":   s.results,
		"scanning":  s.scanning,
		"last_scan": s.lastScan,
	})
}

func (s *Service) handleGetStatus(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"scanning":  s.scanning,
		"status":    s.status,
		"last_scan": s.lastScan,
	})
}
