package ais

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
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
	"github.com/kaandikec/sdr/internal/ws"
)

type Service struct {
	mu       sync.RWMutex
	cfg      config.AISConfig
	sdrCfg   config.SDRConfig
	proc     *process.Manager
	database *db.DB
	hub      *ws.Hub
	status   services.Status
	ships    map[string]*db.AISShip
	msgCount int
}

func New(cfg config.AISConfig, sdrCfg config.SDRConfig, proc *process.Manager, database *db.DB, hub *ws.Hub) *Service {
	return &Service{
		cfg:      cfg,
		sdrCfg:   sdrCfg,
		proc:     proc,
		database: database,
		hub:      hub,
		status:   services.StatusStopped,
		ships:    make(map[string]*db.AISShip),
	}
}

func (s *Service) ID() string { return "ais" }

func (s *Service) Info() services.ServiceInfo {
	return services.ServiceInfo{
		ID:          "ais",
		Name:        "AIS Gemi Takip",
		Description: "Deniz trafiği izleme sistemi",
		Category:    services.CategoryNative,
		Status:      s.Status(),
		Icon:        "ship",
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

	gain := fmt.Sprintf("%d", s.cfg.Gain)

	p, err := s.proc.Start("ais", "rtl_ais", "-g", gain, "-p", "0")
	if err != nil {
		s.status = services.StatusError
		return fmt.Errorf("starting rtl_ais: %w", err)
	}

	s.status = services.StatusRunning

	// Parse AIS output
	go s.parseOutput(p)

	// Cleanup stale ships periodically
	go s.cleanupLoop(ctx)

	log.Printf("[ais] Started")
	return nil
}

func (s *Service) parseOutput(p *process.ManagedProcess) {
	scanner := bufio.NewScanner(p.Stdout)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "!") {
			continue
		}
		s.processNMEA(line)
	}
}

func (s *Service) processNMEA(sentence string) {
	// Simple NMEA AIS parser for position reports
	// Format: !AIVDM,1,1,,A,<payload>,0*<checksum>
	parts := strings.Split(sentence, ",")
	if len(parts) < 7 {
		return
	}

	payload := parts[5]
	if len(payload) < 20 {
		return
	}

	// Decode 6-bit ASCII payload
	bits := decodeSixBit(payload)
	if len(bits) < 150 {
		return
	}

	msgType := bitsToInt(bits, 0, 6)
	if msgType < 1 || msgType > 3 {
		return // Only handle position reports for now
	}

	mmsi := fmt.Sprintf("%d", bitsToInt(bits, 8, 30))
	status := bitsToInt(bits, 38, 4)
	_ = status

	rot := bitsToInt(bits, 42, 8)
	_ = rot

	sog := float64(bitsToInt(bits, 46, 10)) / 10.0
	lon := float64(bitsToSignedInt(bits, 61, 28)) / 600000.0
	lat := float64(bitsToSignedInt(bits, 89, 27)) / 600000.0
	cog := float64(bitsToInt(bits, 116, 12)) / 10.0
	hdg := bitsToInt(bits, 128, 9)

	if lat < -90 || lat > 90 || lon < -180 || lon > 180 {
		return
	}
	if lat == 0 && lon == 0 {
		return
	}

	now := time.Now()
	ship := &db.AISShip{
		MMSI:     mmsi,
		Lat:      lat,
		Lon:      lon,
		Speed:    sog,
		Course:   cog,
		Heading:  hdg,
		LastSeen: now,
	}

	s.mu.Lock()
	existing, ok := s.ships[mmsi]
	if ok {
		ship.Name = existing.Name
		ship.ShipType = existing.ShipType
		ship.Destination = existing.Destination
		ship.FirstSeen = existing.FirstSeen
	} else {
		ship.FirstSeen = now
	}
	s.ships[mmsi] = ship
	s.msgCount++
	s.mu.Unlock()

	// Save to DB
	s.database.Exec(
		`INSERT INTO ais_ships (mmsi, name, ship_type, destination, last_lat, last_lon, speed, course, heading, last_seen, first_seen)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(mmsi) DO UPDATE SET last_lat=?, last_lon=?, speed=?, course=?, heading=?, last_seen=?`,
		ship.MMSI, ship.Name, ship.ShipType, ship.Destination, lat, lon, sog, cog, hdg, now, ship.FirstSeen,
		lat, lon, sog, cog, hdg, now,
	)

	// Broadcast
	s.hub.Broadcast("ais.ships", ship)
}

func (s *Service) cleanupLoop(ctx context.Context) {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.mu.Lock()
			cutoff := time.Now().Add(-10 * time.Minute)
			for mmsi, ship := range s.ships {
				if ship.LastSeen.Before(cutoff) {
					delete(s.ships, mmsi)
				}
			}
			s.mu.Unlock()
		}
	}
}

func (s *Service) Stop(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.proc.Stop("ais")
	s.status = services.StatusStopped
	return nil
}

func (s *Service) RegisterRoutes(r chi.Router) {
	r.Route("/api/v1/ais", func(r chi.Router) {
		r.Get("/ships", s.handleGetShips)
		r.Get("/stats", s.handleGetStats)
	})
}

func (s *Service) handleGetShips(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	ships := make([]*db.AISShip, 0, len(s.ships))
	for _, ship := range s.ships {
		ships = append(ships, ship)
	}
	stats := map[string]any{
		"total_messages": s.msgCount,
		"active_ships":   len(s.ships),
	}
	s.mu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"ships": ships, "stats": stats})
}

func (s *Service) handleGetStats(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"total_messages": s.msgCount,
		"active_ships":   len(s.ships),
		"status":         s.status,
	})
}

// AIS 6-bit ASCII decoding helpers

func decodeSixBit(payload string) []int {
	bits := make([]int, 0, len(payload)*6)
	for _, c := range payload {
		v := int(c) - 48
		if v > 40 {
			v -= 8
		}
		for i := 5; i >= 0; i-- {
			bits = append(bits, (v>>i)&1)
		}
	}
	return bits
}

func bitsToInt(bits []int, start, length int) int {
	if start+length > len(bits) {
		return 0
	}
	val := 0
	for i := 0; i < length; i++ {
		val = (val << 1) | bits[start+i]
	}
	return val
}

func bitsToSignedInt(bits []int, start, length int) int {
	val := bitsToInt(bits, start, length)
	if bits[start] == 1 {
		val -= 1 << length
	}
	return val
}

// Unused but available for future NMEA parsing
var _ = strconv.Atoi
