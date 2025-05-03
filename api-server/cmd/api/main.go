package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"api-server/internal/server"
)

// serverInterface defines the methods we use from http.Server
// This interface makes testing easier
type serverInterface interface {
	ListenAndServe() error
	Shutdown(ctx context.Context) error
}

// ServerConfig holds configuration for the server
type ServerConfig struct {
	ShutdownTimeout time.Duration
}

// DefaultServerConfig provides default configuration values
func DefaultServerConfig() ServerConfig {
	return ServerConfig{
		ShutdownTimeout: 5 * time.Second,
	}
}

// waitForSignalFunc defines function type for waiting for signals
type waitForSignalFunc func() context.Context

// waitForSignal is the default implementation that waits for interrupt signals
var waitForSignal = func() context.Context {
	// Don't call stop() immediately, as it will cancel notification handling
	// The defer stop() was causing the context to be canceled immediately
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	
	// Store the stop function without calling it
	// The stop function will be called when the context is canceled by a signal
	go func() {
		<-ctx.Done()
		stop()
	}()
	
	return ctx
}

// gracefulShutdown handles the server shutdown process
func gracefulShutdown(apiServer serverInterface, done chan bool, config ServerConfig, ctx context.Context) {
	// Listen for the interrupt signal.
	<-ctx.Done()

	log.Println("shutting down gracefully, press Ctrl+C again to force")

	// The context is used to inform the server it has configured time to finish
	// the request it is currently handling
	shutdownCtx, cancel := context.WithTimeout(context.Background(), config.ShutdownTimeout)
	defer cancel()
	
	if err := apiServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("Server forced to shutdown with error: %v", err)
	}

	log.Println("Server exiting")

	// Notify the main goroutine that the shutdown is complete
	done <- true
}

// runServerFunc defines function type for running the server
type runServerFunc func(srv serverInterface, config ServerConfig, signalCtx context.Context) error

// runServer is the default implementation for starting the server and handling shutdown
var runServer = func(srv serverInterface, config ServerConfig, signalCtx context.Context) error {
	// Create a done channel to signal when the shutdown is complete
	done := make(chan bool, 1)

	// Run graceful shutdown in a separate goroutine
	go gracefulShutdown(srv, done, config, signalCtx)

	// Start the server
	err := srv.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("http server error: %s", err)
	}

	// Wait for the graceful shutdown to complete
	<-done
	log.Println("Graceful shutdown complete.")
	return nil
}

func main() {
	srv := server.NewServer()
	config := DefaultServerConfig()
	signalCtx := waitForSignal()
	
	// Run server and handle errors
	if err := runServer(srv, config, signalCtx); err != nil {
		log.Fatal(err)
	}
}