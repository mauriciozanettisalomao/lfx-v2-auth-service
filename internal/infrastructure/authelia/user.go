// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package authelia

import (
	"context"
	"log/slog"

	"github.com/linuxfoundation/lfx-v2-auth-service/internal/domain/model"
	"github.com/linuxfoundation/lfx-v2-auth-service/internal/domain/port"
	"github.com/linuxfoundation/lfx-v2-auth-service/internal/infrastructure/nats"
	"github.com/linuxfoundation/lfx-v2-auth-service/pkg/constants"
	"github.com/linuxfoundation/lfx-v2-auth-service/pkg/errors"
	"github.com/linuxfoundation/lfx-v2-auth-service/pkg/redaction"
)

// userWriter implements UserReaderWriter with pluggable storage and ConfigMap sync
type userWriter struct {
	sync         *sync
	storage      internalStorageReaderWriter
	orchestrator internalOrchestrator
}

// SearchUser searches for a user in storage
func (a *userWriter) SearchUser(ctx context.Context, user *model.User, criteria string) (*model.User, error) {

	if user == nil {
		return nil, errors.NewValidation("user is required")
	}

	param := func(criteriaType string) string {
		switch criteriaType {
		case constants.CriteriaTypeEmail:
			slog.DebugContext(ctx, "searching user",
				"criteria", criteria,
				"email", redaction.RedactEmail(user.PrimaryEmail),
			)
			return a.storage.BuildLookupKey(ctx, "email", user.BuildEmailIndexKey(ctx))
		case constants.CriteriaTypeUsername:
			slog.DebugContext(ctx, "searching user",
				"criteria", criteria,
				"username", redaction.Redact(user.Username),
			)
			return user.Username
		}
		return ""
	}

	key := param(criteria)
	if key == "" {
		return nil, errors.NewValidation("invalid criteria type")
	}

	existingUser, err := a.storage.GetUser(ctx, key)
	if err != nil {
		slog.ErrorContext(ctx, "failed to get existing user from storage",
			"error", err,
			"key", key,
		)
		return nil, err
	}
	return existingUser.User, nil

}

// GetUser retrieves a user from storage
func (a *userWriter) GetUser(ctx context.Context, user *model.User) (*model.User, error) {
	return nil, errors.NewUnexpected("not implemented")
}

// UpdateUser updates a user only in storage with patch-like behavior, updating only changed fields
func (a *userWriter) UpdateUser(ctx context.Context, user *model.User) (*model.User, error) {
	if user == nil {
		return nil, errors.NewValidation("user is required")
	}

	if user.Username == "" {
		return nil, errors.NewValidation("username is required")
	}

	// TODO: Get the 'sub' from the /api/oidc/userinfo and persist it in the user
	// It requires the token to be validated with the userinfo endpoint

	// First, get the existing user from storage to preserve Authelia-specific fields
	existingAutheliaUser := &AutheliaUser{}
	existingAutheliaUser.SetUsername(user.Username)

	existingUser, err := a.storage.GetUser(ctx, existingAutheliaUser.Username)
	if err != nil {
		slog.ErrorContext(ctx, "failed to get existing user from storage",
			"username", user.Username,
			"error", err,
			"key", existingAutheliaUser.Username,
		)
		return nil, errors.NewUnexpected("failed to get existing user from storage", err)
	}

	// Update UserMetadata if provided - patch individual metadata fields
	if user.UserMetadata != nil {
		if existingUser.UserMetadata == nil {
			existingUser.UserMetadata = &model.UserMetadata{}
		}

		metadataUpdated := existingUser.UserMetadata.Patch(user.UserMetadata)
		if metadataUpdated {
			_, err = a.storage.SetUser(ctx, existingUser)
			if err != nil {
				slog.ErrorContext(ctx, "failed to update user in storage",
					"username", user.Username,
					"error", err,
				)
				return nil, errors.NewUnexpected("failed to update user in storage", err)
			}
		}

	}

	slog.InfoContext(ctx, "user updated successfully in storage",
		"username", user.Username)

	return existingUser.User, nil
}

// NewUserReaderWriter creates a new Authelia User repository
func NewUserReaderWriter(ctx context.Context, config map[string]string, natsClient *nats.NATSClient) (port.UserReaderWriter, error) {
	// Set defaults in case of not set

	u := &userWriter{
		sync: &sync{},
	}

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

	errSyncUsers := u.sync.syncUsers(ctx, u.storage, u.orchestrator)
	if errSyncUsers != nil {
		slog.Warn("failed to sync from storage to orchestrator", "error", errSyncUsers)
	}

	return u, nil
}
