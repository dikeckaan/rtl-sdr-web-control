package server

import (
	"encoding/json"
	"io/fs"
	"log"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/kaandikec/sdr/internal/auth"
	"github.com/kaandikec/sdr/internal/config"
	"github.com/kaandikec/sdr/internal/db"
	"github.com/kaandikec/sdr/internal/services"
	"github.com/kaandikec/sdr/internal/ws"
)

type Server struct {
	cfg      *config.Config
	router   chi.Router
	db       *db.DB
	auth     *auth.Manager
	hub      *ws.Hub
	registry *services.Registry
}

func New(cfg *config.Config, database *db.DB, authMgr *auth.Manager, hub *ws.Hub, registry *services.Registry) *Server {
	s := &Server{
		cfg:      cfg,
		db:       database,
		auth:     authMgr,
		hub:      hub,
		registry: registry,
	}

	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Compress(5))
	r.Use(corsMiddleware)
	r.Use(authMgr.Middleware)

	// WebSocket endpoint
	r.HandleFunc("/ws", hub.HandleWS)

	// Auth routes
	r.Post("/api/v1/auth/login", s.handleLogin)
	r.Post("/api/v1/auth/logout", s.handleLogout)
	r.Get("/api/v1/auth/me", s.handleMe)
	r.Get("/api/v1/auth/setup-check", s.handleSetupCheck)
	r.Post("/api/v1/auth/setup", s.handleSetup)

	// Service management routes
	r.Get("/api/v1/services", s.handleListServices)
	r.Post("/api/v1/services/{id}/start", s.handleStartService)
	r.Post("/api/v1/services/{id}/stop", s.handleStopService)
	r.Post("/api/v1/services/{id}/restart", s.handleRestartService)
	r.Post("/api/v1/services/stop-all", s.handleStopAll)

	// System routes
	r.Get("/api/v1/system/status", s.handleSystemStatus)

	s.router = r
	return s
}

func (s *Server) Router() chi.Router {
	return s.router
}

// RegisterServiceRoutes adds service-specific routes to the router.
func (s *Server) RegisterServiceRoutes(svc services.Service) {
	svc.RegisterRoutes(s.router)
}

// ServeSPA serves the embedded frontend SPA.
func (s *Server) ServeSPA(frontendFS fs.FS) {
	fileServer := http.FileServer(http.FS(frontendFS))

	s.router.HandleFunc("/*", func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/")
		if path == "" {
			path = "index.html"
		}

		f, err := frontendFS.Open(path)
		if err != nil {
			r.URL.Path = "/"
			fileServer.ServeHTTP(w, r)
			return
		}
		f.Close()

		fileServer.ServeHTTP(w, r)
	})
}

// ServeFrontend serves the frontend. During development, it serves from the filesystem.
// In production, it uses the embedded frontend.
func (s *Server) ServeFrontend() {
	// Try to serve from web/frontend/dist if it exists (development)
	distDir := "web/frontend/dist"
	if info, err := os.Stat(distDir); err == nil && info.IsDir() {
		log.Printf("[server] Serving frontend from %s", distDir)
		s.router.HandleFunc("/*", func(w http.ResponseWriter, r *http.Request) {
			path := distDir + r.URL.Path
			if _, err := os.Stat(path); os.IsNotExist(err) {
				http.ServeFile(w, r, distDir+"/index.html")
				return
			}
			http.ServeFile(w, r, path)
		})
		return
	}

	// Fallback: serve a minimal built-in dashboard
	log.Printf("[server] No frontend dist found, serving built-in dashboard")
	s.router.HandleFunc("/*", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") || strings.HasPrefix(r.URL.Path, "/ws") {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(builtinDashboard))
	})
}

func (s *Server) Start() error {
	log.Printf("[server] Listening on %s", s.cfg.Server.Listen)
	return http.ListenAndServe(s.cfg.Server.Listen, s.router)
}

// Handlers

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}

	user, err := s.db.GetUser(req.Username)
	if err != nil || user == nil || !auth.CheckPassword(user.PasswordHash, req.Password) {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid credentials"})
		return
	}

	token := s.auth.CreateSession(user.Username, user.Role)

	http.SetCookie(w, &http.Cookie{
		Name:     "sdr_session",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   86400,
	})

	writeJSON(w, http.StatusOK, map[string]any{
		"token":    token,
		"username": user.Username,
		"role":     user.Role,
	})
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	cookie, _ := r.Cookie("sdr_session")
	if cookie != nil {
		s.auth.DestroySession(cookie.Value)
	}

	http.SetCookie(w, &http.Cookie{
		Name:   "sdr_session",
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	})

	writeJSON(w, http.StatusOK, map[string]string{"status": "logged out"})
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"username": r.Header.Get("X-User"),
		"role":     r.Header.Get("X-Role"),
	})
}

func (s *Server) handleSetupCheck(w http.ResponseWriter, r *http.Request) {
	count, _ := s.db.UserCount()
	writeJSON(w, http.StatusOK, map[string]any{
		"setup_required": count == 0,
	})
}

func (s *Server) handleSetup(w http.ResponseWriter, r *http.Request) {
	count, _ := s.db.UserCount()
	if count > 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "setup already completed"})
		return
	}

	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Username == "" || req.Password == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "username and password required"})
		return
	}

	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "hash error"})
		return
	}

	if err := s.db.CreateUser(req.Username, hash, "admin"); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	// Auto-login after setup
	token := s.auth.CreateSession(req.Username, "admin")
	http.SetCookie(w, &http.Cookie{
		Name:     "sdr_session",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   86400,
	})

	writeJSON(w, http.StatusOK, map[string]any{
		"status":   "setup complete",
		"token":    token,
		"username": req.Username,
		"role":     "admin",
	})
}

func (s *Server) handleListServices(w http.ResponseWriter, r *http.Request) {
	svcs := s.registry.All()
	writeJSON(w, http.StatusOK, map[string]any{
		"services":       svcs,
		"active_service": s.registry.ActiveID(),
	})
}

func (s *Server) handleStartService(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := s.registry.Start(r.Context(), id); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "started", "service": id})
}

func (s *Server) handleStopService(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := s.registry.Stop(r.Context(), id); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "stopped", "service": id})
}

func (s *Server) handleRestartService(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := s.registry.Stop(r.Context(), id); err != nil {
		log.Printf("[server] Restart stop error: %v", err)
	}
	time.Sleep(2 * time.Second)
	if err := s.registry.Start(r.Context(), id); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "restarted", "service": id})
}

func (s *Server) handleStopAll(w http.ResponseWriter, r *http.Request) {
	s.registry.StopAll(r.Context())
	writeJSON(w, http.StatusOK, map[string]string{"status": "all stopped"})
}

func (s *Server) handleSystemStatus(w http.ResponseWriter, r *http.Request) {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	hostname, _ := os.Hostname()

	writeJSON(w, http.StatusOK, map[string]any{
		"hostname":       hostname,
		"active_service": s.registry.ActiveID(),
		"uptime_go":      time.Since(startTime).String(),
		"goroutines":     runtime.NumGoroutine(),
		"memory_mb":      memStats.Alloc / 1024 / 1024,
		"ws_clients":     s.hub.ClientCount(),
		"station": map[string]any{
			"name": s.cfg.Station.Name,
			"lat":  s.cfg.Station.Latitude,
			"lon":  s.cfg.Station.Longitude,
			"alt":  s.cfg.Station.Altitude,
		},
	})
}

var startTime = time.Now()

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
