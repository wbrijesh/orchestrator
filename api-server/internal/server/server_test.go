package server

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v4"

	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"api-server/internal/browser"
	"api-server/internal/database"
)

var (
	containerTeardown func(context.Context, ...testcontainers.TerminateOption) error
	apiServerStop     context.CancelFunc

	// dynamically set in TestMain
	apiBaseURL string

	jwtSecret = []byte("test-secret")
)

// TestMain sets up external dependencies (Postgres, Browser stub, JWT overrides)
// and tears them down after all tests.
func TestMain(m *testing.M) {
	// 1. Start Postgres if not already provided
	if os.Getenv("CI_DB_PROVIDED") != "true" {
		td, err := startPostgresContainer()
		if err != nil {
			log.Fatalf("could not start postgres container: %v", err)
		}
		containerTeardown = td
	}

	// 2. Apply DB migrations
	dbSvc, err := database.New()
	if err != nil {
		log.Fatalf("failed to create database service: %v", err)
	}
	if err := applyMigrations(dbSvc.DB()); err != nil {
		log.Fatalf("failed to apply migrations: %v", err)
	}

	// Determine Browser server availability or create stub
	ensureBrowserServer()

	// Determine API server base URL. If external server running (env TEST_API_BASE_URL or localhost default), use it. Otherwise start our own.
	apiBaseURL = ensureAPIServer(dbSvc)

	// 5. Execute tests
	code := m.Run()

	// teardown
	if apiServerStop != nil {
		apiServerStop()
	}
	_ = dbSvc.Close()
	if containerTeardown != nil {
		_ = containerTeardown(context.Background())
	}
	os.Exit(code)
}

/******************************* Helpers ********************************/

// startPostgresContainer starts a temporary Postgres container and populates the
// DB_* env vars used by database.New().
func startPostgresContainer() (func(context.Context, ...testcontainers.TerminateOption) error, error) {
	const (
		dbName = "api"
		dbUser = "user"
		dbPwd  = "password"
	)

	pgC, err := postgres.Run(
		context.Background(),
		"postgres:latest",
		postgres.WithDatabase(dbName),
		postgres.WithUsername(dbUser),
		postgres.WithPassword(dbPwd),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").WithOccurrence(2).WithStartupTimeout(30*time.Second),
		),
	)
	if err != nil {
		return nil, err
	}

	host, err := pgC.Host(context.Background())
	if err != nil {
		return pgC.Terminate, err
	}
	port, err := pgC.MappedPort(context.Background(), "5432/tcp")
	if err != nil {
		return pgC.Terminate, err
	}

	os.Setenv("DB_HOST", host)
	os.Setenv("DB_PORT", port.Port())
	os.Setenv("DB_USERNAME", dbUser)
	os.Setenv("DB_PASSWORD", dbPwd)
	os.Setenv("DB_DATABASE", dbName)

	return pgC.Terminate, nil
}

// applyMigrations runs all *.up.sql scripts under migrations/.
func applyMigrations(db *sql.DB) error {
	migrationsDir := filepath.Join("..", "..", "migrations") // relative to internal/server/
	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		return err
	}
	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".up.sql") {
			files = append(files, filepath.Join(migrationsDir, e.Name()))
		}
	}
	sort.Strings(files)
	for _, f := range files {
		content, err := os.ReadFile(f)
		if err != nil {
			return err
		}
		if _, err := db.Exec(string(content)); err != nil {
			return err
		}
	}
	return nil
}

// jwtSign signs claims using test secret.
func jwtSign(claims map[string]any) (string, error) {
	header := map[string]any{"alg": "HS256", "typ": "JWT"}
	hJSON, _ := json.Marshal(header)
	cJSON, _ := json.Marshal(claims)
	seg := func(b []byte) string { return strings.TrimRight(base64URLEncode(b), "=") }
	unsigned := seg(hJSON) + "." + seg(cJSON)
	sig := hmacSHA256([]byte(unsigned), jwtSecret)
	return unsigned + "." + seg(sig), nil
}

func jwtValidate(tkn string) (*jwt.RegisteredClaims, error) {
	// basic validation for tests: split and ignore
	parts := strings.Split(tkn, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid token")
	}
	payload, err := base64URLDecode(parts[1])
	if err != nil {
		return nil, err
	}
	var claims jwt.RegisteredClaims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, err
	}
	return &claims, nil
}

// base64URLEncode encodes bytes using base64 URL encoding without padding.
func base64URLEncode(b []byte) string {
	return base64.RawURLEncoding.EncodeToString(b)
}

// base64URLDecode decodes a base64 URL encoded string without padding.
func base64URLDecode(s string) ([]byte, error) {
	return base64.RawURLEncoding.DecodeString(s)
}

// hmacSHA256 returns HMAC-SHA256 of message with the given key.
func hmacSHA256(message, key []byte) []byte {
	mac := hmac.New(sha256.New, key)
	mac.Write(message)
	return mac.Sum(nil)
}

/******************************* Tests **********************************/

type apiResp struct {
	Error string          `json:"error"`
	Data  json.RawMessage `json:"data"`
}

type authData struct {
	Token string `json:"token"`
	User  struct {
		ID string `json:"id"`
	} `json:"user"`
}

type sessionData struct {
	Session struct {
		ID string `json:"id"`
	} `json:"session"`
}

func TestHelloWorld(t *testing.T) {
	resp := mustRequest(t, http.MethodGet, "/", nil, "")
	require.Contains(t, string(resp), "Hello World")
}

func TestAuthAndSessionFlow(t *testing.T) {
	// 1. Register
	regBody := database.AuthRequest{Email: "flow@example.com", Password: "secret", FirstName: "F", LastName: "L"}
	regJSON, _ := json.Marshal(regBody)
	regRespRaw := mustRequest(t, http.MethodPost, "/register", bytes.NewReader(regJSON), "")
	var regEnvelope apiResp
	require.NoError(t, json.Unmarshal(regRespRaw, &regEnvelope))
	require.Empty(t, regEnvelope.Error)
	var regData authData
	require.NoError(t, json.Unmarshal(regEnvelope.Data, &regData))
	token := regData.Token
	userID := regData.User.ID
	require.NotEmpty(t, token)
	require.NotEmpty(t, userID)

	// 2. Login
	loginJSON, _ := json.Marshal(database.AuthRequest{Email: regBody.Email, Password: regBody.Password})
	loginRespRaw := mustRequest(t, http.MethodPost, "/login", bytes.NewReader(loginJSON), "")
	var loginEnv apiResp
	require.NoError(t, json.Unmarshal(loginRespRaw, &loginEnv))
	require.Empty(t, loginEnv.Error)

	// 3. Create session
	sessRaw := mustRequest(t, http.MethodPost, "/sessions", nil, token)
	var sessEnv apiResp
	require.NoError(t, json.Unmarshal(sessRaw, &sessEnv))
	require.Empty(t, sessEnv.Error)
	var sessData sessionData
	require.NoError(t, json.Unmarshal(sessEnv.Data, &sessData))
	sessID := sessData.Session.ID
	require.NotEmpty(t, sessID)

	// 4. List sessions
	listRaw := mustRequest(t, http.MethodGet, "/sessions", nil, token)
	require.Contains(t, string(listRaw), sessID)

	// 5. Stop session
	stopRaw := mustRequest(t, http.MethodPost, "/sessions/"+sessID+"/stop", nil, token)
	require.Contains(t, string(stopRaw), "stopped_at")

	// 6. Delete session
	delRaw := mustRequest(t, http.MethodDelete, "/sessions/"+sessID, nil, token)
	require.Contains(t, string(delRaw), "success")
}

/******************************* Request util ***************************/

func mustRequest(t *testing.T, method, path string, body io.Reader, token string) []byte {
	t.Helper()
	req, err := http.NewRequest(method, apiBaseURL+path, body)
	require.NoError(t, err)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	out, _ := io.ReadAll(resp.Body)
	require.Equal(t, http.StatusOK, resp.StatusCode, string(out))
	return out
}

// newBrowserStubOnPort spins up an http.Server listening on desired port that
// implements minimal subset of the Browser service contract needed for tests.
func newBrowserStubOnPort(portStr string) *http.Server {
	mux := http.NewServeMux()

	mux.HandleFunc("/sessions", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		var req browser.CreateSessionRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		now := time.Now()
		resp := browser.SessionResponse{
			ID:          "stub-session-" + req.BrowserType,
			BrowserType: req.BrowserType,
			Headless:    req.Headless,
			CreatedAt:   browser.FlexibleTime(now),
			ExpiresAt:   browser.FlexibleTime(now.Add(1 * time.Hour)),
			CdpURL:      "ws://stub",
			ViewportSize: browser.ViewportSize{
				Width:  req.ViewportSize.Width,
				Height: req.ViewportSize.Height,
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	})

	mux.HandleFunc("/sessions/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
	})

	srv := &http.Server{Addr: ":" + portStr, Handler: mux}
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("browser stub failed: %v", err)
		}
	}()
	return srv
}

// ensureBrowserServer verifies if a browser server is already running; if not, spins a stub and sets env.
func ensureBrowserServer() {
	browserURL := os.Getenv("BROWSER_SERVER_URL")
	if browserURL == "" {
		browserURL = "http://localhost:8000"
	}

	if serverResponds(browserURL+"/sessions", http.MethodGet) { // simple reachability check
		return
	}

	// need to start stub on random free port
	l, err := net.Listen("tcp", ":0")
	if err != nil {
		log.Fatalf("failed to find free port for browser stub: %v", err)
	}
	portStr := fmt.Sprintf("%d", l.Addr().(*net.TCPAddr).Port)
	_ = l.Close()

	stub := newBrowserStubOnPort(portStr)
	// store cancel via apiServerStop if needed later
	apiServerStop = func() { _ = stub.Close() }
	os.Setenv("BROWSER_SERVER_URL", "http://localhost:"+portStr)
}

// ensureAPIServer returns base URL; may start internal server if external not reachable
func ensureAPIServer(dbSvc database.Service) string {
	ext := os.Getenv("TEST_API_BASE_URL")
	if ext == "" {
		ext = "http://localhost:8080"
	}
	if serverResponds(ext+"/health", http.MethodGet) {
		return ext
	}

	// Start internal HTTP test server on random port
	srv := httptest.NewServer((&Server{db: dbSvc}).RegisterRoutes())
	apiServerStop = srv.Close
	return srv.URL
}

// serverResponds checks if GET to url returns 200
func serverResponds(url, method string) bool {
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return false
	}
	client := http.Client{Timeout: 1 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// testEnvOrDefault helper for tests (duplicate of server util but local)
func testEnvOrDefault(key, defaultVal string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return defaultVal
}
