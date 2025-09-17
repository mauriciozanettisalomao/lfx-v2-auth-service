// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/linuxfoundation/lfx-v2-auth-service/cmd/server/service"

	authservice "github.com/linuxfoundation/lfx-v2-auth-service/gen/auth_service"
	logging "github.com/linuxfoundation/lfx-v2-auth-service/pkg/log"
)

const (
	defaultPort = "8080"
	// gracefulShutdownSeconds should be higher than NATS client
	// request timeout, and lower than the pod or liveness probe's
	// terminationGracePeriodSeconds.
	gracefulShutdownSeconds = 25
)

func init() {
	// slog is the standard library logger, we use it to log errors and
	logging.InitStructureLogConfig()
}

func main() {
	// Define command line flags
	var (
		dbgF = flag.Bool("d", false, "enable debug logging")
		port = flag.String("p", defaultPort, "listen port")
		bind = flag.String("bind", "*", "interface to bind on")
	)
	flag.Usage = func() {
		flag.PrintDefaults()
		os.Exit(2)
	}
	flag.Parse()

	ctx := context.Background()
	slog.InfoContext(ctx, "Starting auth service",
		"bind", *bind,
		"http-port", *port,
		"graceful-shutdown-seconds", gracefulShutdownSeconds,
	)

	// Initialize the auth service
	authSvc := service.NewAuthService()

	// Wrap the service in endpoints
	authEndpoints := authservice.NewEndpoints(authSvc)

	// Create channel for shutdown signals
	errc := make(chan error, 1)

	// Setup interrupt handler
	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
		errc <- fmt.Errorf("%s", <-c)
	}()

	var wg sync.WaitGroup
	ctx, cancel := context.WithCancel(ctx)

	// Start the HTTP server for auth service
	addr := ":" + *port
	if *bind != "*" {
		addr = *bind + ":" + *port
	}

	handleHTTPServer(ctx, addr, authEndpoints, &wg, errc, *dbgF)

	// TODO: Start NATS subscriptions here if needed
	// if err := startNATSSubscriptions(ctx); err != nil {
	//     slog.ErrorContext(ctx, "failed to start NATS subscriptions", "error", err)
	//     errc <- fmt.Errorf("failed to start NATS subscriptions: %w", err)
	// }

	// Wait for signal.
	slog.InfoContext(ctx, "received shutdown signal, stopping servers",
		"signal", <-errc,
	)

	// Send cancellation signal to the goroutines.
	cancel()

	// Create a timeout context for graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), gracefulShutdownSeconds*time.Second)
	defer shutdownCancel()

	// Wait for all goroutines to finish with timeout
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		slog.InfoContext(ctx, "graceful shutdown completed")
	case <-shutdownCtx.Done():
		slog.WarnContext(ctx, "graceful shutdown timed out")
	}

	slog.InfoContext(ctx, "exited")
}

// handleHTTPServer starts the HTTP server for auth service endpoints
func handleHTTPServer(ctx context.Context, host string, authEndpoints *authservice.Endpoints, wg *sync.WaitGroup, errc chan<- error, dbg bool) {
	// Implementation would be similar to cmd/server/http.go
	// For now, this is a placeholder - the actual server is in cmd/server/main.go
	wg.Add(1)
	go func() {
		defer wg.Done()
		// TODO: Implement HTTP server handling
		slog.InfoContext(ctx, "HTTP server placeholder", "host", host)
	}()
}
