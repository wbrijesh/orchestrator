package server

import (
	"api-server/internal/database"
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
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
	rand.Seed(time.Now().UnixNano())
	adj := adjectives[rand.Intn(len(adjectives))]
	noun := nouns[rand.Intn(len(nouns))]
	return fmt.Sprintf("%s-%s-%d", adj, noun, rand.Intn(1000))
}

// Request and response structures
type CreateSessionResponse struct {
	Session *database.SessionView `json:"session"`
}

type SessionsResponse struct {
	Sessions []*database.SessionView `json:"sessions"`
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

	// Create session in database
	ctx := context.Background()
	session, err := s.db.CreateSession(ctx, userID, sessionName)
	if err != nil {
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
			Session: session.ToView(),
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

	// Stop session in database
	ctx := context.Background()
	session, err := s.db.StopSession(ctx, sessionID, userID)
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
			Session: session.ToView(),
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

	// Delete session from database
	ctx := context.Background()
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