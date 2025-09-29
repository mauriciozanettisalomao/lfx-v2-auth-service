// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package authelia

import (
	"context"
	"log/slog"

	"github.com/linuxfoundation/lfx-v2-auth-service/internal/domain/model"
	"github.com/linuxfoundation/lfx-v2-auth-service/internal/domain/port"
	"github.com/linuxfoundation/lfx-v2-auth-service/internal/infrastructure/nats"
	"github.com/linuxfoundation/lfx-v2-auth-service/pkg/errors"
)

// userWriter implements UserReaderWriter with pluggable storage and ConfigMap sync
type userWriter struct {
	storage      internalStorageReaderWriter
	orchestrator internalOrchestrator
}

// SearchUser searches for a user in storage
func (a *userWriter) SearchUser(ctx context.Context, user *model.User, criteria string) (*model.User, error) {
	return nil, errors.NewUnexpected("not implemented")
}

// GetUser retrieves a user from storage
func (a *userWriter) GetUser(ctx context.Context, user *model.User) (*model.User, error) {
	return nil, errors.NewUnexpected("not implemented")
}

// UpdateUser updates a user in storage and syncs to ConfigMap
func (a *userWriter) UpdateUser(ctx context.Context, user *model.User) (*model.User, error) {
	return nil, errors.NewUnexpected("not implemented")
}

// NewUserReaderWriter creates a new Authelia User repository
func NewUserReaderWriter(ctx context.Context, config map[string]string, natsClient *nats.NATSClient) (port.UserReaderWriter, error) {
	// Set defaults in case of not set

	u := &userWriter{}

	// Initialize storage using NATS KV store
	if u.storage == nil {
		storage, errNATSUserStorage := newNATSUserStorage(ctx, natsClient)
		if errNATSUserStorage != nil {
			slog.ErrorContext(ctx, "failed to create storage", "error", errNATSUserStorage)
			return nil, errNATSUserStorage
		}
		u.storage = storage
	}

	// Initialize orchestrator using K8S to update the ConfigMap, Secrets and DaemonSet
	if u.orchestrator == nil {
		orchestrator, errK8sOrchestrator := newK8sUserOrchestrator(ctx, config)
		if errK8sOrchestrator != nil {
			slog.ErrorContext(ctx, "failed to create orchestrator", "error", errK8sOrchestrator)
			return nil, errK8sOrchestrator
		}
		u.orchestrator = orchestrator
	}

	// errSyncUsers := u.syncUsers(ctx)
	// if errSyncUsers != nil {
	// 	slog.Warn("failed to sync from storage to orchestrator", "error", errSyncUsers)
	// }

	return u, nil
}
