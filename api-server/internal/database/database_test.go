package database

import (
    "context"
    "database/sql"
    "log"
    "os"
    "path/filepath"
    "sort"
    "strings"
    "testing"
    "time"

    "github.com/google/uuid"
    "github.com/stretchr/testify/require"
    "github.com/testcontainers/testcontainers-go"
    "github.com/testcontainers/testcontainers-go/modules/postgres"
    "github.com/testcontainers/testcontainers-go/wait"
)

// Container teardown callback (set in TestMain).
var containerTeardown func(context.Context, ...testcontainers.TerminateOption) error

// shared database service used across tests to avoid closing the singleton
var sharedSvc Service

// TestMain starts a postgres test-container, runs migrations and executes the
// package tests. When running inside a CI environment that already provisions a
// database, the container step is skipped by setting CI_DB_PROVIDED=true.
func TestMain(m *testing.M) {
    if os.Getenv("CI_DB_PROVIDED") != "true" {
        td, err := startPostgresContainer()
        if err != nil {
            log.Fatalf("could not start postgres container: %v", err)
        }
        containerTeardown = td
    }

    // Apply migrations so that the schema exists before tests run.
    var err error
    sharedSvc, err = New()
    if err != nil {
        log.Fatalf("failed to connect DB in TestMain: %v", err)
    }
    if err := applyMigrations(sharedSvc.DB()); err != nil {
        log.Fatalf("failed to apply migrations: %v", err)
    }
    // DO NOT close here; keep connection for the entire test run

    code := m.Run()

    if containerTeardown != nil {
        _ = containerTeardown(context.Background())
    }
    if sharedSvc != nil {
        _ = sharedSvc.Close()
    }
    os.Exit(code)
}

// startPostgresContainer provisions a postgres container and sets the DB_* env
// vars consumed by database.New().
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
    // DB_SCHEMA intentionally left empty (defaults to public).

    return pgC.Terminate, nil
}

// applyMigrations executes all *.up.sql files under migrations/ in lexical
// order using the provided connection. This avoids an additional dependency on
// a migration library and keeps the test environment self-contained.
func applyMigrations(db *sql.DB) error {
    migrationsDir := filepath.Join("..", "..", "migrations") // relative to internal/database/
    entries, err := os.ReadDir(migrationsDir)
    if err != nil {
        return err
    }

    // select *.up.sql files and sort them for deterministic order
    var files []string
    for _, e := range entries {
        if e.IsDir() {
            continue
        }
        name := e.Name()
        if strings.HasSuffix(name, ".up.sql") {
            files = append(files, filepath.Join(migrationsDir, name))
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

// TestHealth verifies the Health implementation returns an "up" status when
// the database is reachable.
func TestHealth(t *testing.T) {
    dbSvc := mustDB(t)

    stats := dbSvc.Health()
    require.Equal(t, "up", stats["status"], "expected status to be up")
    require.NotContains(t, stats, "error", "expected no error field in health response")
}

// TestUserQueries exercises CreateUser and GetUserByEmail.
func TestUserQueries(t *testing.T) {
    dbSvc := mustDB(t)

    ctx := context.Background()
    user := &User{
        Email:        "test@example.com",
        FirstName:    "Test",
        LastName:     "User",
        PasswordHash: "hashed-password",
    }

    id, err := dbSvc.CreateUser(ctx, user)
    require.NoError(t, err)
    require.NotEqual(t, uuid.Nil, id)

    fetched, err := dbSvc.GetUserByEmail(ctx, user.Email)
    require.NoError(t, err)
    require.Equal(t, id, fetched.ID)
    require.Equal(t, user.FirstName, fetched.FirstName)
}

// TestSessionQueries covers create, retrieve, stop and delete session flows.
func TestSessionQueries(t *testing.T) {
    dbSvc := mustDB(t)
    ctx := context.Background()

    // A user is required for FK constraint.
    user := &User{
        Email:        "session@example.com",
        FirstName:    "Sess",
        LastName:     "Ion",
        PasswordHash: "hashed",
    }
    userID, err := dbSvc.CreateUser(ctx, user)
    require.NoError(t, err)

    // 1. Create session
    sess, err := dbSvc.CreateSession(ctx, userID, "first-session", "browser-id", "firefox", "ws://cdp", false, 1280, 720, nil)
    require.NoError(t, err)
    require.Equal(t, "first-session", sess.Name)
    require.False(t, sess.StoppedAt.Valid)

    // 2. Get by ID
    same, err := dbSvc.GetSessionByID(ctx, sess.ID, userID)
    require.NoError(t, err)
    require.Equal(t, sess.ID, same.ID)

    // 3. List by user
    list, err := dbSvc.GetSessionsByUserID(ctx, userID)
    require.NoError(t, err)
    require.Len(t, list, 1)

    // 4. Stop session
    stopped, err := dbSvc.StopSession(ctx, sess.ID, userID)
    require.NoError(t, err)
    require.True(t, stopped.StoppedAt.Valid)

    // 5. Delete session
    require.NoError(t, dbSvc.DeleteSession(ctx, sess.ID, userID))

    // 6. Ensure it is gone
    _, err = dbSvc.GetSessionByID(ctx, sess.ID, userID)
    require.Error(t, err)
}

// mustDB is a helper that returns a ready Service instance or fails the test.
func mustDB(t *testing.T) Service {
    t.Helper()
    if sharedSvc != nil {
        return sharedSvc
    }
    dbSvc, err := New()
    require.NoError(t, err)
    return dbSvc
}
