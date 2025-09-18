// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package service

import (
	"context"
	"encoding/json"

	"github.com/linuxfoundation/lfx-v2-auth-service/internal/domain/model"
	"github.com/linuxfoundation/lfx-v2-auth-service/internal/domain/port"
	"github.com/linuxfoundation/lfx-v2-auth-service/pkg/errors"
)

// messageHandlerOrchestrator orchestrates the message handling process
type messageHandlerOrchestrator struct {
	userWriter UserServiceWriter
}

// messageHandlerOrchestratorOption defines a function type for setting options
type messageHandlerOrchestratorOption func(*messageHandlerOrchestrator)

// WithUserWriterForMessageHandler sets the user writer for the message handler orchestrator
func WithUserWriterForMessageHandler(userWriter UserServiceWriter) messageHandlerOrchestratorOption {
	return func(m *messageHandlerOrchestrator) {
		m.userWriter = userWriter
	}
}

// UpdateUser updates the user in the identity provider
func (m *messageHandlerOrchestrator) UpdateUser(ctx context.Context, msg port.TransportMessenger) ([]byte, error) {

	user := &model.User{}
	err := json.Unmarshal(msg.Data(), user)
	if err != nil {
		return nil, errors.NewUnexpected("failed to unmarshal user", err)
	}

	// It's calling another service to update the user because in case of
	// need to expose the same functionality using another pattern, like http rest,
	// we can do without changing the user writer orchestrator
	updatedUser, err := m.userWriter.UpdateUser(ctx, user)
	if err != nil {
		return nil, errors.NewUnexpected("failed to update user", err)
	}
	// TODO: handle the response from the user writer
	updatedUserJSON, err := json.Marshal(updatedUser)
	if err != nil {
		return nil, errors.NewUnexpected("failed to marshal updated user", err)
	}
	return updatedUserJSON, nil
}

// NewMessageHandlerOrchestrator creates a new message handler orchestrator using the option pattern
func NewMessageHandlerOrchestrator(opts ...messageHandlerOrchestratorOption) port.MessageHandler {
	m := &messageHandlerOrchestrator{}
	for _, opt := range opts {
		opt(m)
	}
	return m
}
