package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type Manager struct {
	secret   []byte
	sessions sync.Map // token -> Session
}

type Session struct {
	Username  string    `json:"username"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

func NewManager(secret string) *Manager {
	return &Manager{secret: []byte(secret)}
}

func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(hash), err
}

func CheckPassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

func (m *Manager) CreateSession(username, role string) string {
	b := make([]byte, 32)
	rand.Read(b)
	token := hex.EncodeToString(b)

	mac := hmac.New(sha256.New, m.secret)
	mac.Write([]byte(token))
	signed := hex.EncodeToString(mac.Sum(nil))

	fullToken := token + "." + signed

	m.sessions.Store(fullToken, Session{
		Username:  username,
		Role:      role,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
	})

	return fullToken
}

func (m *Manager) ValidateSession(token string) (*Session, bool) {
	val, ok := m.sessions.Load(token)
	if !ok {
		return nil, false
	}
	sess := val.(Session)
	if time.Now().After(sess.ExpiresAt) {
		m.sessions.Delete(token)
		return nil, false
	}
	return &sess, true
}

func (m *Manager) DestroySession(token string) {
	m.sessions.Delete(token)
}

func (m *Manager) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip auth for login, setup, and setup-check endpoints
		if r.URL.Path == "/api/v1/auth/login" || r.URL.Path == "/api/v1/auth/setup" || r.URL.Path == "/api/v1/auth/setup-check" {
			next.ServeHTTP(w, r)
			return
		}
		if !strings.HasPrefix(r.URL.Path, "/api/") && !strings.HasPrefix(r.URL.Path, "/ws") {
			next.ServeHTTP(w, r)
			return
		}

		token := extractToken(r)
		if token == "" {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "authentication required"})
			return
		}

		sess, valid := m.ValidateSession(token)
		if !valid {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid or expired session"})
			return
		}

		r.Header.Set("X-User", sess.Username)
		r.Header.Set("X-Role", sess.Role)
		next.ServeHTTP(w, r)
	})
}

func extractToken(r *http.Request) string {
	// Check Authorization header
	auth := r.Header.Get("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimPrefix(auth, "Bearer ")
	}

	// Check cookie
	cookie, err := r.Cookie("sdr_session")
	if err == nil {
		return cookie.Value
	}

	// Check query param (for WebSocket)
	return r.URL.Query().Get("token")
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func GenerateDefaultHash() {
	hash, _ := HashPassword("admin")
	fmt.Println(hash)
}
