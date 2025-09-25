// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package service

import (
	"context"

	"github.com/linuxfoundation/lfx-v2-auth-service/internal/domain/model"
	"github.com/linuxfoundation/lfx-v2-auth-service/internal/domain/port"
)

// UserServiceReader defines the behavior of the user service reader
type UserServiceReader interface {
	GetUser(ctx context.Context, user *model.User) (*model.User, error)
	SearchUser(ctx context.Context, user *model.User, criteria string) (*model.User, error)
}

// userReaderOrchestrator orchestrates the user reader process
type userReaderOrchestrator struct {
	userReader port.UserReader
}

// userReaderOrchestratorOption defines the option for the user reader orchestrator
type userReaderOrchestratorOption func(*userReaderOrchestrator)

// WithUserReader sets the user reader for the user reader orchestrator
func WithUserReader(userReader port.UserReader) userReaderOrchestratorOption {
	return func(o *userReaderOrchestrator) {
		o.userReader = userReader
	}
}

// GetUser retrieves the user from the identity provider
func (u *userReaderOrchestrator) GetUser(ctx context.Context, user *model.User) (*model.User, error) {
	return u.userReader.GetUser(ctx, user)
}

// SearchUser searches the user from the identity provider
func (u *userReaderOrchestrator) SearchUser(ctx context.Context, user *model.User, criteria string) (*model.User, error) {
	return u.userReader.SearchUser(ctx, user, criteria)
}

// NewUserReaderOrchestrator creates a new user writer orchestrator
func NewUserReaderOrchestrator(opts ...userReaderOrchestratorOption) UserServiceReader {
	o := &userReaderOrchestrator{}
	for _, opt := range opts {
		opt(o)
	}
	return o
}
