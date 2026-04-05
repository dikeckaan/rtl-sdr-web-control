package pager

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"regexp"
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

var msgRegex = regexp.MustCompile(`(POCSAG\d+|FLEX|FLEX_SECURE):\s*Address:\s*(\d+)\s*Function:\s*(\d+)\s*(?:Alpha|Numeric|Tone):\s*(.*)`)

type Service struct {
	mu       sync.RWMutex
	cfg      config.PagerConfig
	sdrCfg   config.SDRConfig
	proc     *process.Manager
	database *db.DB
	hub      *ws.Hub
	status   services.Status
	freq     string
	messages []db.PagerMessage
}

func New(cfg config.PagerConfig, sdrCfg config.SDRConfig, proc *process.Manager, database *db.DB, hub *ws.Hub) *Service {
	return &Service{
		cfg:      cfg,
		sdrCfg:   sdrCfg,
		proc:     proc,
		database: database,
		hub:      hub,
		status:   services.StatusStopped,
		freq:     cfg.DefaultFreq,
		messages: make([]db.PagerMessage, 0, 100),
	}
}

func (s *Service) ID() string { return "pager" }

func (s *Service) Info() services.ServiceInfo {
	return services.ServiceInfo{
		ID:          "pager",
		Name:        "Pager / POCSAG",
		Description: "POCSAG/FLEX çağrı cihazı çözücü",
		Category:    services.CategoryNative,
		Status:      s.Status(),
		Icon:        "message-square",
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

	// Build multimon-ng protocol args
	protoArgs := ""
	for _, p := range s.cfg.Protocols {
		protoArgs += fmt.Sprintf("-a %s ", p)
	}
	if protoArgs == "" {
		protoArgs = "-a POCSAG512 -a POCSAG1200 -a POCSAG2400 -a FLEX"
	}

	gain := fmt.Sprintf("%d", s.sdrCfg.Gain)
	cmdStr := fmt.Sprintf("rtl_fm -M fm -f %s -s 22050 -g %s - | multimon-ng -t raw %s -f alpha -", s.freq, gain, strings.TrimSpace(protoArgs))

	p, err := s.proc.Start("pager", "sh", "-c", cmdStr)
	if err != nil {
		s.status = services.StatusError
		return fmt.Errorf("starting pager: %w", err)
	}

	s.status = services.StatusRunning

	// Parse output in background
	go s.parseOutput(p)

	log.Printf("[pager] Started on %s", s.freq)
	return nil
}

func (s *Service) parseOutput(p *process.ManagedProcess) {
	scanner := bufio.NewScanner(p.Stdout)
	for scanner.Scan() {
		line := scanner.Text()
		msg := s.parseLine(line)
		if msg == nil {
			continue
		}

		s.mu.Lock()
		s.messages = append(s.messages, *msg)
		if len(s.messages) > 1000 {
			s.messages = s.messages[len(s.messages)-500:]
		}
		s.mu.Unlock()

		// Save to DB
		s.database.Exec(
			"INSERT INTO pager_messages (timestamp, msg_type, address, func_code, message, frequency, raw_line) VALUES (?, ?, ?, ?, ?, ?, ?)",
			msg.Timestamp, msg.MsgType, msg.Address, msg.FuncCode, msg.Message, msg.Frequency, msg.RawLine,
		)

		// Broadcast via WebSocket
		s.hub.Broadcast("pager.messages", msg)
	}
}

func (s *Service) parseLine(line string) *db.PagerMessage {
	matches := msgRegex.FindStringSubmatch(line)
	if matches == nil {
		return nil
	}

	funcCode := 0
	fmt.Sscanf(matches[3], "%d", &funcCode)

	return &db.PagerMessage{
		Timestamp: time.Now(),
		MsgType:   matches[1],
		Address:   matches[2],
		FuncCode:  funcCode,
		Message:   strings.TrimSpace(matches[4]),
		Frequency: s.freq,
		RawLine:   line,
	}
}

func (s *Service) Stop(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.proc.Stop("pager")
	s.status = services.StatusStopped
	return nil
}

func (s *Service) RegisterRoutes(r chi.Router) {
	r.Route("/api/v1/pager", func(r chi.Router) {
		r.Get("/state", s.handleGetState)
		r.Get("/messages", s.handleGetMessages)
		r.Post("/freq", s.handleSetFreq)
	})
}

func (s *Service) handleGetState(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"freq":          s.freq,
		"status":        s.status,
		"message_count": len(s.messages),
	})
}

func (s *Service) handleGetMessages(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	limit := 100
	msgs := s.messages
	if len(msgs) > limit {
		msgs = msgs[len(msgs)-limit:]
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"messages": msgs})
}

func (s *Service) handleSetFreq(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Freq string `json:"freq"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Freq == "" {
		http.Error(w, `{"error":"freq required"}`, http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	wasRunning := s.status == services.StatusRunning
	s.mu.Unlock()

	if wasRunning {
		s.Stop(r.Context())
	}

	s.mu.Lock()
	s.freq = req.Freq
	s.mu.Unlock()

	if wasRunning {
		s.Start(r.Context())
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"freq": req.Freq, "status": s.Status()})
}
