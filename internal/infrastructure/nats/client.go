// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package nats

import (
	"context"
	"log/slog"
	"time"

	"github.com/linuxfoundation/lfx-v2-auth-service/internal/domain/port"
	"github.com/linuxfoundation/lfx-v2-auth-service/pkg/constants"
	"github.com/linuxfoundation/lfx-v2-auth-service/pkg/errors"

	"github.com/nats-io/nats.go"
)

// NATSClient wraps the NATS connection and provides access control operations
type NATSClient struct {
	conn    *nats.Conn
	config  Config
	timeout time.Duration
}

// NATSClientInterface defines the interface for NATS operations
// This allows for easy mocking and testing
type NATSClientInterface interface {
	Close() error
	IsReady(ctx context.Context) error
}

// Close gracefully closes the NATS connection
func (c *NATSClient) Close() error {
	if c.conn != nil {
		c.conn.Close()
	}
	return nil
}

// IsReady checks if the NATS client is ready
func (c *NATSClient) IsReady(ctx context.Context) error {
	if c.conn == nil {
		return errors.NewServiceUnavailable("NATS client is not initialized or not connected")
	}
	if !c.conn.IsConnected() || c.conn.IsDraining() {
		return errors.NewServiceUnavailable("NATS client is not ready, connection is not established or is draining")
	}
	return nil
}

// SubscribeWithTransportMessenger subscribes to a subject with proper TransportMessenger handling
func (c *NATSClient) SubscribeWithTransportMessenger(ctx context.Context, subject string, queueName string, handler func(context.Context, port.TransportMessenger)) (*nats.Subscription, error) {
	return c.conn.QueueSubscribe(subject, queueName, func(msg *nats.Msg) {
		transportMsg := NewTransportMessenger(msg)

		defer func() {
			if r := recover(); r != nil {
				slog.ErrorContext(ctx, "panic in NATS handler",
					"subject", subject,
					"queue", queueName,
					"panic", r,
				)
			}
		}()

		handler(ctx, transportMsg)
	})
}

// NewClient creates a new NATS client with the given configuration
func NewClient(ctx context.Context, config Config) (*NATSClient, error) {
	slog.InfoContext(ctx, "creating NATS client",
		"url", config.URL,
		"timeout", config.Timeout,
	)

	// Validate configuration
	if config.URL == "" {
		return nil, errors.NewUnexpected("NATS URL is required")
	}

	// Configure NATS connection options
	opts := []nats.Option{
		nats.Name(constants.ServiceName),
		nats.Timeout(config.Timeout),
		nats.MaxReconnects(config.MaxReconnect),
		nats.ReconnectWait(config.ReconnectWait),
		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			slog.WarnContext(ctx, "NATS disconnected", "error", err)
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			slog.InfoContext(ctx, "NATS reconnected", "url", nc.ConnectedUrl())
		}),
		nats.ErrorHandler(func(_ *nats.Conn, s *nats.Subscription, err error) {
			if s != nil {
				slog.With("error", err, "subject", s.Subject, "queue", s.Queue).Error("async NATS error")
			} else {
				slog.With("error", err).Error("async NATS error outside subscription")
			}
		}),
		nats.ClosedHandler(func(nc *nats.Conn) {
			slog.InfoContext(ctx, "NATS connection closed")
		}),
	}

	// Establish connection
	conn, err := nats.Connect(config.URL, opts...)
	if err != nil {
		return nil, errors.NewServiceUnavailable("failed to connect to NATS", err)
	}

	client := &NATSClient{
		conn:    conn,
		config:  config,
		timeout: config.Timeout,
	}

	slog.InfoContext(ctx, "NATS client created successfully",
		"connected_url", conn.ConnectedUrl(),
		"status", conn.Status(),
	)

	return client, nil
}
