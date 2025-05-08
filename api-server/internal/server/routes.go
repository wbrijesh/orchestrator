package server

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

func (s *Server) RegisterRoutes() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Logger)

	// Add New Relic middleware if available
	if s.nrApp != nil {
		r.Use(s.NewRelicMiddleware)
	}

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"https://*", "http://*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// miscellaneous
	r.Get("/", s.HelloWorldHandler)
	r.Get("/health", s.healthHandler)

	// public
	r.Post("/register", s.RegisterHandler)
	r.Post("/login", s.LoginHandler)

	// protected routes
	r.Group(func(r chi.Router) {
		// Apply auth middleware to this group
		r.Use(s.AuthMiddleware)

		// Session routes
		r.Post("/sessions", s.CreateSessionHandler)
		r.Get("/sessions", s.GetUserSessionsHandler)
		r.Post("/sessions/{id}/stop", s.StopSessionHandler)
		r.Delete("/sessions/{id}", s.DeleteSessionHandler)
	})

	return r
}

func (s *Server) HelloWorldHandler(w http.ResponseWriter, r *http.Request) {
	resp := make(map[string]string)
	resp["message"] = "Hello World"

	jsonResp, err := json.Marshal(resp)
	if err != nil {
		log.Fatalf("error handling JSON marshal. Err: %v", err)
	}

	_, _ = w.Write(jsonResp)
}

func (s *Server) healthHandler(w http.ResponseWriter, r *http.Request) {
	jsonResp, _ := json.Marshal(s.db.Health())
	_, _ = w.Write(jsonResp)
}
