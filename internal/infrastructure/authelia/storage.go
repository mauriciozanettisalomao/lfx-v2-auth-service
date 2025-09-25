// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package authelia

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/linuxfoundation/lfx-v2-auth-service/internal/infrastructure/nats"
	"github.com/linuxfoundation/lfx-v2-auth-service/pkg/constants"
	"github.com/linuxfoundation/lfx-v2-auth-service/pkg/errors"
	"github.com/nats-io/nats.go/jetstream"
)

// storageReaderWriter defines the interface for user storage operations
// It works with AutheliaUser internally for proper Authelia-specific handling
type storageReaderWriter interface {
	GetUser(ctx context.Context, username string) (*AutheliaUser, error)
	SetUser(ctx context.Context, username string, user *AutheliaUser) error
	ListUsers(ctx context.Context) (map[string]*AutheliaUser, error)
}

// natsUserStorage implements UserStorage using NATS KV store
type natsUserStorage struct {
	natsClient *nats.NATSClient
	kvStore    jetstream.KeyValue
}

// GetUser retrieves a user from NATS KV store
func (n *natsUserStorage) GetUser(ctx context.Context, username string) (*AutheliaUser, error) {
	entry, err := n.kvStore.Get(ctx, username)
	if err != nil {
		if err == jetstream.ErrKeyNotFound {
			return nil, errors.NewNotFound("user not found")
		}
		return nil, errors.NewUnexpected("failed to get user from NATS KV", err)
	}

	var autheliaUser AutheliaUser
	if err := json.Unmarshal(entry.Value(), &autheliaUser); err != nil {
		return nil, errors.NewUnexpected("failed to unmarshal user data", err)
	}

	return &autheliaUser, nil
}

// SetUser stores a user in NATS KV store
func (n *natsUserStorage) SetUser(ctx context.Context, username string, user *AutheliaUser) error {
	// Ensure username is set in the embedded model.User
	if user.User != nil {
		if user.User.Username == "" {
			user.User.Username = username
		}
		if user.User.UserID == "" {
			user.User.UserID = username
		}
	}

	// Update timestamp
	user.UpdatedAt = time.Now()

	// If this is a new user (no CreatedAt), set it
	if user.CreatedAt.IsZero() {
		user.CreatedAt = time.Now()
	}

	data, err := json.Marshal(user)
	if err != nil {
		return errors.NewUnexpected("failed to marshal user data", err)
	}

	_, err = n.kvStore.Put(ctx, username, data)
	if err != nil {
		return errors.NewUnexpected("failed to store user in NATS KV", err)
	}

	return nil
}

// ListUsers retrieves all users from NATS KV store
func (n *natsUserStorage) ListUsers(ctx context.Context) (map[string]*AutheliaUser, error) {
	users := make(map[string]*AutheliaUser)

	// Get all keys from the KV store
	keys, err := n.kvStore.Keys(ctx)
	if err != nil {
		return nil, errors.NewUnexpected("failed to list keys from NATS KV", err)
	}

	// Retrieve each user
	for _, key := range keys {
		user, err := n.GetUser(ctx, key)
		if err != nil {
			slog.WarnContext(ctx, "failed to get user during list operation",
				"username", key, "error", err)
			continue
		}
		users[key] = user
	}

	return users, nil
}

// newNATSUserStorage creates a new NATS-based user storage
func newNATSUserStorage(ctx context.Context, natsClient *nats.NATSClient) (storageReaderWriter, error) {
	// Get the KV store for authelia users
	kvStore, exists := natsClient.GetKVStore(constants.KVBucketNameAutheliaUsers)
	if !exists {
		return nil, errors.NewUnexpected("authelia users KV bucket not found in NATS client")
	}

	slog.DebugContext(ctx, "created NATS user storage", "kvStore", kvStore)

	return &natsUserStorage{
		natsClient: natsClient,
		kvStore:    kvStore,
	}, nil
}
