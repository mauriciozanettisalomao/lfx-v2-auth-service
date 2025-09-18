// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package service

import (
	"context"

	"github.com/linuxfoundation/lfx-v2-auth-service/internal/domain/model"
	"github.com/linuxfoundation/lfx-v2-auth-service/internal/domain/port"
)

// UserServiceWriter defines the behavior of the user service writer
type UserServiceWriter interface {
	UpdateUser(ctx context.Context, user *model.User) (*model.User, error)
}

// userWriterOrchestrator orchestrates the user writer process
type userWriterOrchestrator struct {
	userWriter port.UserWriter
}

// CommitteeWriter defines the interface for committee write operations
type userWriterOrchestratorOption func(*userWriterOrchestrator)

// WithUserWriter sets the user writer for the user writer orchestrator
func WithUserWriter(userWriter port.UserWriter) userWriterOrchestratorOption {
	return func(o *userWriterOrchestrator) {
		o.userWriter = userWriter
	}
}

// UpdateUser updates the user in the identity provider
func (u *userWriterOrchestrator) UpdateUser(ctx context.Context, user *model.User) (*model.User, error) {

	// TODO: perform validation/sanitization of the user data

	return u.userWriter.UpdateUser(ctx, user)
}

// NewUserWriterOrchestrator creates a new user writer orchestrator
func NewUserWriterOrchestrator(opts ...userWriterOrchestratorOption) UserServiceWriter {
	o := &userWriterOrchestrator{}
	for _, opt := range opts {
		opt(o)
	}
	return o
}
