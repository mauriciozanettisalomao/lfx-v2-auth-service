// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package service

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/linuxfoundation/lfx-v2-auth-service/internal/domain/port"
	"github.com/linuxfoundation/lfx-v2-auth-service/internal/infrastructure/auth0"
	"github.com/linuxfoundation/lfx-v2-auth-service/internal/infrastructure/mock"
	"github.com/linuxfoundation/lfx-v2-auth-service/internal/infrastructure/nats"
	"github.com/linuxfoundation/lfx-v2-auth-service/internal/service"
	"github.com/linuxfoundation/lfx-v2-auth-service/pkg/constants"
)

var (
	// expose the NATS client for direct access in subscriptions
	natsClient *nats.NATSClient

	natsDoOnce sync.Once
)

func natsInit(ctx context.Context) {

	natsDoOnce.Do(func() {
		natsURL := os.Getenv("NATS_URL")
		if natsURL == "" {
			natsURL = "nats://localhost:4222"
		}

		natsTimeout := os.Getenv("NATS_TIMEOUT")
		if natsTimeout == "" {
			natsTimeout = "10s"
		}
		natsTimeoutDuration, err := time.ParseDuration(natsTimeout)
		if err != nil {
			log.Fatalf("invalid NATS timeout duration: %v", err)
		}

		natsMaxReconnect := os.Getenv("NATS_MAX_RECONNECT")
		if natsMaxReconnect == "" {
			natsMaxReconnect = "3"
		}
		natsMaxReconnectInt, err := strconv.Atoi(natsMaxReconnect)
		if err != nil {
			log.Fatalf("invalid NATS max reconnect value %s: %v", natsMaxReconnect, err)
		}

		natsReconnectWait := os.Getenv("NATS_RECONNECT_WAIT")
		if natsReconnectWait == "" {
			natsReconnectWait = "2s"
		}
		natsReconnectWaitDuration, err := time.ParseDuration(natsReconnectWait)
		if err != nil {
			log.Fatalf("invalid NATS reconnect wait duration %s : %v", natsReconnectWait, err)
		}

		config := nats.Config{
			URL:           natsURL,
			Timeout:       natsTimeoutDuration,
			MaxReconnect:  natsMaxReconnectInt,
			ReconnectWait: natsReconnectWaitDuration,
		}

		client, errNewClient := nats.NewClient(ctx, config)
		if errNewClient != nil {
			log.Fatalf("failed to create NATS client: %v", errNewClient)
		}
		natsClient = client
	})
}

// newUserReaderWriter creates a UserReaderWriter implementation based on the environment variable.
// Set USER_REPOSITORY_TYPE to "mock" to use the mock implementation, or "auth0" (or leave unset) for Auth0.
func newUserReaderWriter(ctx context.Context) port.UserReaderWriter {
	userRepositoryType := os.Getenv(constants.UserRepositoryTypeEnvKey)
	if userRepositoryType == "" {
		userRepositoryType = constants.UserRepositoryTypeMock // default to mock for development
	}

	switch userRepositoryType {
	case constants.UserRepositoryTypeMock:
		slog.InfoContext(ctx, "using mock user repository implementation")
		return mock.NewUserReaderWriter(ctx)
	case constants.UserRepositoryTypeAuth0:
		slog.InfoContext(ctx, "using Auth0 user repository implementation")
		return auth0.NewUserReaderWriter()
	default:
		slog.WarnContext(ctx, "unknown user repository type, defaulting to Auth0", "type", userRepositoryType)
		return auth0.NewUserReaderWriter()
	}
}

// QueueSubscriptions starts all NATS subscriptions with the provided dependencies
func QueueSubscriptions(ctx context.Context) error {
	slog.InfoContext(ctx, "starting NATS subscriptions")

	// Initialize NATS client first
	natsInit(ctx)

	messageHandlerService := &MessageHandlerService{
		messageHandler: service.NewMessageHandlerOrchestrator(
			service.WithUserWriterForMessageHandler(
				service.NewUserWriterOrchestrator(service.WithUserWriter(newUserReaderWriter(ctx))),
			),
		),
	}

	// Get the NATS client - we need to access it directly
	natsClient := getNATSClient()
	if natsClient == nil {
		return fmt.Errorf("NATS client not initialized")
	}

	// Start subscriptions for each subject
	subjects := map[string]func(context.Context, port.TransportMessenger){
		constants.UserUpdateSubject: messageHandlerService.HandleMessage,
		// Add more subjects here as needed
	}

	for subject, handler := range subjects {
		slog.InfoContext(ctx, "subscribing to NATS subject", "subject", subject)
		if _, err := natsClient.SubscribeWithTransportMessenger(ctx, subject, constants.AuthServiceQueue, handler); err != nil {
			slog.ErrorContext(ctx, "failed to subscribe to NATS subject",
				"error", err,
				"subject", subject,
			)
			return fmt.Errorf("failed to subscribe to subject %s: %w", subject, err)
		}
	}

	slog.InfoContext(ctx, "NATS subscriptions started successfully")
	return nil
}

// getNATSClient returns the initialized NATS client
// This is a helper function to access the client for subscription management
func getNATSClient() *nats.NATSClient {
	return natsClient
}
