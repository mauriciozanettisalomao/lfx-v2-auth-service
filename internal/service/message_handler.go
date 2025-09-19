// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package service

import (
	"context"
	"encoding/json"

	"github.com/linuxfoundation/lfx-v2-auth-service/internal/domain/model"
	"github.com/linuxfoundation/lfx-v2-auth-service/internal/domain/port"
)

// UserUpdateResponse represents the response structure for user update operations
type UserDataResponse struct {
	Success bool   `json:"success"`
	Data    any    `json:"data,omitempty"`
	Error   string `json:"error,omitempty"`
}

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

func (m *messageHandlerOrchestrator) errorResponse(error string) []byte {
	response := UserDataResponse{
		Success: false,
		Error:   error,
	}
	responseJSON, _ := json.Marshal(response)
	return responseJSON
}

// UpdateUser updates the user in the identity provider
func (m *messageHandlerOrchestrator) UpdateUser(ctx context.Context, msg port.TransportMessenger) ([]byte, error) {

	if m.userWriter == nil {
		return m.errorResponse("user service unavailable"), nil
	}

	user := &model.User{}
	err := json.Unmarshal(msg.Data(), user)
	if err != nil {
		responseJSON := m.errorResponse("failed to unmarshal user data")
		return responseJSON, nil
	}

	// It's calling another service to update the user because in case of
	// need to expose the same functionality using another pattern, like http rest,
	// we can do without changing the user writer orchestrator
	updatedUser, err := m.userWriter.UpdateUser(ctx, user)
	if err != nil {
		responseJSON := m.errorResponse(err.Error())
		return responseJSON, nil
	}

	// Return success response with user metadata
	response := UserDataResponse{
		Success: true,
		Data:    updatedUser.UserMetadata,
	}

	responseJSON, err := json.Marshal(response)
	if err != nil {
		errorResponseJSON := m.errorResponse("failed to marshal response")
		return errorResponseJSON, nil
	}

	return responseJSON, nil
}

// NewMessageHandlerOrchestrator creates a new message handler orchestrator using the option pattern
func NewMessageHandlerOrchestrator(opts ...messageHandlerOrchestratorOption) port.MessageHandler {
	m := &messageHandlerOrchestrator{}
	for _, opt := range opts {
		opt(m)
	}
	return m
}
