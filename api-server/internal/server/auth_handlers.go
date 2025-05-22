package server

import (
	"api-server/internal/auth"
	"api-server/internal/database"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

func toView(u *database.User) *database.UserView {
	return &database.UserView{
		ID:        u.ID.String(),
		Email:     u.Email,
		FirstName: u.FirstName,
		LastName:  u.LastName,
	}
}

// RegisterHandler creates a user & returns a JWT.
func (s *Server) RegisterHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Log the incoming request
	log.Printf("RegisterHandler: received request")

	var req database.AuthRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("RegisterHandler: failed to decode request body: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(database.APIResponse{
			Error: "bad request",
			Data:  nil,
		})
		return
	}

	log.Printf("RegisterHandler: processing request for email: %s", req.Email)
	// hash the password
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("RegisterHandler: failed to hash password: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(database.APIResponse{
			Error: "server error",
			Data:  nil,
		})
		return
	}
	u := &database.User{
		Email:        req.Email,
		FirstName:    req.FirstName,
		LastName:     req.LastName,
		PasswordHash: string(hash),
	}

	log.Printf("RegisterHandler: creating user with email: %s", u.Email)

	ctx := context.Background()
	id, err := s.db.CreateUser(ctx, u)
	if err != nil {
		log.Printf("RegisterHandler: failed to create user: %v", err)
		msg := "could not create user"
		if strings.Contains(err.Error(), "duplicate key value") {
			msg = "email already exists"
			w.WriteHeader(http.StatusBadRequest)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
		json.NewEncoder(w).Encode(database.APIResponse{
			Error: msg,
			Data:  nil,
		})
		return
	}
	u.ID = id
	log.Printf("RegisterHandler: successfully created user with ID: %s", u.ID)

	// generate token
	token, err := auth.GenerateToken(u.ID.String())
	if err != nil {
		log.Printf("RegisterHandler: failed to generate token: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(database.APIResponse{
			Error: "could not sign token",
			Data:  nil,
		})
		return
	}
	log.Printf("RegisterHandler: successfully generated token for user: %s", u.Email)

	authResponse := database.AuthResponse{
		Token: token,
		User:  toView(u),
	}

	json.NewEncoder(w).Encode(database.APIResponse{
		Error: "",
		Data:  authResponse,
	})
}

// LoginHandler checks credentials & returns a JWT.
func (s *Server) LoginHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req database.AuthRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(database.APIResponse{
			Error: "bad request",
			Data:  nil,
		})
		return
	}
	ctx := context.Background()
	u, err := s.db.GetUserByEmail(ctx, req.Email)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(database.APIResponse{
			Error: "invalid credentials",
			Data:  nil,
		})
		return
	}
	// compare password
	if bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(req.Password)) != nil {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(database.APIResponse{
			Error: "invalid credentials",
			Data:  nil,
		})
		return
	}
	token, err := auth.GenerateToken(u.ID.String())
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(database.APIResponse{
			Error: "could not sign token",
			Data:  nil,
		})
		return
	}

	authResponse := database.AuthResponse{
		Token: token,
		User:  toView(u),
	}

	json.NewEncoder(w).Encode(database.APIResponse{
		Error: "",
		Data:  authResponse,
	})
}
