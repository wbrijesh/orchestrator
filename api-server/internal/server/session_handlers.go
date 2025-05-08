package server

import (
	"api-server/internal/browser"
	"api-server/internal/database"
	"context"
	"encoding/json"
	"crypto/rand"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// Common word lists for generating session names
var (
	adjectives = []string{
		"autumn", "hidden", "bitter", "misty", "silent", "empty", "dry", "dark",
		"summer", "icy", "delicate", "quiet", "white", "cool", "spring", "winter",
		"patient", "twilight", "dawn", "crimson", "wispy", "weathered", "blue",
		"billowing", "broken", "cold", "damp", "falling", "frosty", "green",
		"long", "late", "lingering", "bold", "little", "morning", "muddy", "old",
		"red", "rough", "still", "small", "sparkling", "throbbing", "shy",
		"wandering", "withered", "wild", "black", "young", "holy", "solitary",
		"fragrant", "aged", "snowy", "proud", "floral", "restless", "divine",
		"polished", "ancient", "purple", "lively", "nameless",
	}
	
	nouns = []string{
		"waterfall", "river", "breeze", "moon", "rain", "wind", "sea", "morning",
		"snow", "lake", "sunset", "pine", "shadow", "leaf", "dawn", "glitter",
		"forest", "hill", "cloud", "meadow", "sun", "glade", "bird", "brook",
		"butterfly", "bush", "dew", "dust", "field", "fire", "flower", "firefly",
		"feather", "grass", "haze", "mountain", "night", "pond", "darkness",
		"snowflake", "silence", "sound", "sky", "shape", "surf", "thunder",
		"violet", "water", "wildflower", "wave", "water", "resonance", "sun",
		"wood", "dream", "cherry", "tree", "fog", "frost", "voice", "paper",
		"frog", "smoke", "star",
	}
)

// RandomSessionName generates a random session name
func RandomSessionName() string {
	// Get a random adjective
	adjIdx, _ := rand.Int(rand.Reader, big.NewInt(int64(len(adjectives))))
	adj := adjectives[adjIdx.Int64()]
	
	// Get a random noun
	nounIdx, _ := rand.Int(rand.Reader, big.NewInt(int64(len(nouns))))
	noun := nouns[nounIdx.Int64()]
	
	// Get a random number between 0-999
	num, _ := rand.Int(rand.Reader, big.NewInt(1000))
	
	return fmt.Sprintf("%s-%s-%d", adj, noun, num.Int64())
}

// Request and response structures
type CreateSessionResponse struct {
	Session *database.SessionView `json:"session"`
}

type SessionsResponse struct {
	Sessions []*database.SessionView `json:"sessions"`
}

// Default browser settings from environment or hardcoded defaults
var (
	defaultBrowserType = getEnvOrDefault("DEFAULT_BROWSER_TYPE", "firefox")
	defaultHeadless    = getEnvBoolOrDefault("DEFAULT_BROWSER_HEADLESS", false)
	defaultViewportW   = getEnvIntOrDefault("DEFAULT_BROWSER_VIEWPORT_WIDTH", 1280)
	defaultViewportH   = getEnvIntOrDefault("DEFAULT_BROWSER_VIEWPORT_HEIGHT", 720)
	defaultTimeout     = getEnvIntOrDefault("DEFAULT_BROWSER_TIMEOUT", 3600) // Default 1 hour
)

// Helper functions to get environment variables with defaults
func getEnvOrDefault(key, defaultVal string) string {
	if val, exists := os.LookupEnv(key); exists {
		return val
	}
	return defaultVal
}

func getEnvBoolOrDefault(key string, defaultVal bool) bool {
	if val, exists := os.LookupEnv(key); exists {
		return val == "true" || val == "1" || val == "yes"
	}
	return defaultVal
}

func getEnvIntOrDefault(key string, defaultVal int) int {
	if val, exists := os.LookupEnv(key); exists {
		if intVal, err := parseInt(val); err == nil {
			return intVal
		}
	}
	return defaultVal
}

func parseInt(s string) (int, error) {
	var i int
	_, err := fmt.Sscanf(s, "%d", &i)
	return i, err
}

// CreateSessionHandler creates a new session for the authenticated user
func (s *Server) CreateSessionHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Get user ID from context (set by AuthMiddleware)
	userID, err := GetUserIDFromContext(r.Context())
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Generate a random session name
	sessionName := RandomSessionName()

	// Create browser client
	browserClient := browser.NewClient()

	// Create browser session request
	browserReq := browser.CreateSessionRequest{
		BrowserType: defaultBrowserType,
		Headless:    defaultHeadless,
		ViewportSize: &browser.ViewportSize{
			Width:  defaultViewportW,
			Height: defaultViewportH,
		},
		Timeout: &defaultTimeout,
	}

	// Create browser session
	ctx := r.Context()
	browserSession, err := browserClient.CreateSession(ctx, browserReq)
	if err != nil {
		log.Printf("Failed to create browser session: %v", err)
		
		// Check if we should use a fallback mock session
		if os.Getenv("BROWSER_SERVER_FALLBACK") == "true" {
			log.Printf("Using fallback mock session")
			// Create a mock browser session for fallback
			now := time.Now()
			expireTime := now.Add(time.Duration(defaultTimeout) * time.Second)
			browserSession = &browser.SessionResponse{
				ID:          "mock-" + uuid.New().String(),
				BrowserType: browserReq.BrowserType,
				Headless:    browserReq.Headless,
				CreatedAt:   browser.FlexibleTime(now),
				ExpiresAt:   browser.FlexibleTime(expireTime),
				CdpURL:      "mock://browser-session/not-available",
				ViewportSize: browser.ViewportSize{
					Width:  browserReq.ViewportSize.Width,
					Height: browserReq.ViewportSize.Height,
				},
				UserAgent: browserReq.UserAgent,
			}
		} else {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(database.APIResponse{
				Error: "Could not create browser session",
				Data:  nil,
			})
			return
		}
	}

	// Create database session using browser session details
	var userAgentPtr *string = nil
	if browserSession.UserAgent != nil {
		userAgentPtr = browserSession.UserAgent
	}

	dbSession, err := s.db.CreateSession(
		ctx, 
		userID, 
		sessionName,
		browserSession.ID, 
		browserSession.BrowserType, 
		browserSession.CdpURL,
		browserSession.Headless,
		browserSession.ViewportSize.Width,
		browserSession.ViewportSize.Height,
		userAgentPtr,
	)
	if err != nil {
		// Try to cleanup the browser session
		_ = browserClient.DeleteSession(ctx, browserSession.ID)

		log.Printf("Failed to record session in database: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(database.APIResponse{
			Error: "Could not create session",
			Data:  nil,
		})
		return
	}

	// Return success response
	json.NewEncoder(w).Encode(database.APIResponse{
		Error: "",
		Data: CreateSessionResponse{
			Session: dbSession.ToView(),
		},
	})
}

// GetUserSessionsHandler retrieves all sessions for the authenticated user
func (s *Server) GetUserSessionsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Get user ID from context (set by AuthMiddleware)
	userID, err := GetUserIDFromContext(r.Context())
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get sessions from database
	ctx := context.Background()
	sessions, err := s.db.GetSessionsByUserID(ctx, userID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(database.APIResponse{
			Error: "Could not retrieve sessions",
			Data:  nil,
		})
		return
	}

	// Convert to session views
	sessionViews := make([]*database.SessionView, 0, len(sessions))
	for _, session := range sessions {
		sessionViews = append(sessionViews, session.ToView())
	}

	// Return success response
	json.NewEncoder(w).Encode(database.APIResponse{
		Error: "",
		Data: SessionsResponse{
			Sessions: sessionViews,
		},
	})
}

// StopSessionHandler stops an active session
func (s *Server) StopSessionHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Get user ID from context (set by AuthMiddleware)
	userID, err := GetUserIDFromContext(r.Context())
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get session ID from URL
	sessionIDStr := chi.URLParam(r, "id")
	if strings.TrimSpace(sessionIDStr) == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(database.APIResponse{
			Error: "Session ID is required",
			Data:  nil,
		})
		return
	}

	// Parse session ID
	sessionID, err := uuid.Parse(sessionIDStr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(database.APIResponse{
			Error: "Invalid session ID",
			Data:  nil,
		})
		return
	}

	// Get session first to get browser ID
	ctx := r.Context()
	session, err := s.db.GetSessionByID(ctx, sessionID, userID)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(database.APIResponse{
			Error: err.Error(),
			Data:  nil,
		})
		return
	}

	// Stop session in browser server if it's active
	if !session.StoppedAt.Valid && session.BrowserID != "" {
		browserClient := browser.NewClient()
		if err := browserClient.DeleteSession(ctx, session.BrowserID); err != nil {
			// Log but continue - we still want to mark the session as stopped
			log.Printf("Failed to stop browser session %s: %v", session.BrowserID, err)
		}
	}

	// Stop session in database
	stoppedSession, err := s.db.StopSession(ctx, sessionID, userID)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(database.APIResponse{
			Error: err.Error(),
			Data:  nil,
		})
		return
	}

	// Return success response
	json.NewEncoder(w).Encode(database.APIResponse{
		Error: "",
		Data: CreateSessionResponse{
			Session: stoppedSession.ToView(),
		},
	})
}

// DeleteSessionHandler permanently deletes a session
func (s *Server) DeleteSessionHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Get user ID from context (set by AuthMiddleware)
	userID, err := GetUserIDFromContext(r.Context())
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get session ID from URL
	sessionIDStr := chi.URLParam(r, "id")
	if strings.TrimSpace(sessionIDStr) == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(database.APIResponse{
			Error: "Session ID is required",
			Data:  nil,
		})
		return
	}

	// Parse session ID
	sessionID, err := uuid.Parse(sessionIDStr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(database.APIResponse{
			Error: "Invalid session ID",
			Data:  nil,
		})
		return
	}

	// Get session first to get browser ID
	ctx := r.Context()
	session, err := s.db.GetSessionByID(ctx, sessionID, userID)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(database.APIResponse{
			Error: err.Error(),
			Data:  nil,
		})
		return
	}

	// Delete session in browser server if it exists
	if session.BrowserID != "" {
		browserClient := browser.NewClient()
		if err := browserClient.DeleteSession(ctx, session.BrowserID); err != nil {
			// Log but continue - we still want to delete the database record
			log.Printf("Failed to delete browser session %s: %v", session.BrowserID, err)
		}
	}

	// Delete session from database
	err = s.db.DeleteSession(ctx, sessionID, userID)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(database.APIResponse{
			Error: err.Error(),
			Data:  nil,
		})
		return
	}

	// Return success response
	json.NewEncoder(w).Encode(database.APIResponse{
		Error: "",
		Data:  map[string]bool{"success": true},
	})
}