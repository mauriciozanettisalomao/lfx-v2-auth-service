// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package authelia

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/linuxfoundation/lfx-v2-auth-service/internal/infrastructure/nats"
	"github.com/linuxfoundation/lfx-v2-auth-service/pkg/constants"
	errs "github.com/linuxfoundation/lfx-v2-auth-service/pkg/errors"
	"github.com/nats-io/nats.go/jetstream"
)

type internalStorageReaderWriter interface {
	GetUser(ctx context.Context, user *AutheliaUser) (*AutheliaUser, error)
	ListUsers(ctx context.Context) (map[string]*AutheliaUser, error)
	SetUser(ctx context.Context, user *AutheliaUser) (any, error)
}

// natsUserStorage implements UserStorage using NATS KV store
type natsUserStorage struct {
	natsClient *nats.NATSClient
	kvStore    jetstream.KeyValue
}

func (n *natsUserStorage) GetUser(ctx context.Context, user *AutheliaUser) (*AutheliaUser, error) {

	if user == nil || user.Username == "" {
		return nil, errs.NewUnexpected("user is required")
	}

	entry, err := n.kvStore.Get(ctx, user.Username)
	if err != nil {
		if errors.Is(err, jetstream.ErrKeyNotFound) {
			return nil, errs.NewNotFound("user not found")
		}
		return nil, errs.NewUnexpected("failed to get user from NATS KV", err)
	}

	var storageUser AutheliaUserStorage
	if err := json.Unmarshal(entry.Value(), &storageUser); err != nil {
		return nil, errs.NewUnexpected("failed to unmarshal user data", err)
	}

	// Convert storage format back to AutheliaUser
	var autheliaUser AutheliaUser
	autheliaUser.FromStorage(&storageUser)

	return &autheliaUser, nil
}

func (n *natsUserStorage) ListUsers(ctx context.Context) (map[string]*AutheliaUser, error) {
	users := make(map[string]*AutheliaUser)

	// Get all keys from the KV store
	keys, err := n.kvStore.Keys(ctx)
	if err != nil && !strings.Contains(err.Error(), "no keys found") {
		return nil, errs.NewUnexpected("failed to list keys from NATS KV", err)
	}

	// Retrieve each user
	for _, key := range keys {
		autheliaUser := &AutheliaUser{}
		autheliaUser.SetUsername(key)
		user, err := n.GetUser(ctx, autheliaUser)
		if err != nil {
			slog.WarnContext(ctx, "failed to get user during list operation",
				"username", key, "error", err)
			continue
		}
		users[key] = user
	}

	return users, nil
}

func (n *natsUserStorage) SetUser(ctx context.Context, user *AutheliaUser) (any, error) {

	// Update timestamp
	user.UpdatedAt = time.Now()

	// If this is a new user (no CreatedAt), set it
	if user.CreatedAt.IsZero() {
		user.CreatedAt = time.Now()
	}

	// Convert to storage format (excludes sensitive fields)
	storageUser := user.ToStorage()

	data, err := json.Marshal(storageUser)
	if err != nil {
		return nil, errs.NewUnexpected("failed to marshal user data", err)
	}

	return n.kvStore.Put(ctx, user.Username, data)
}

// newNATSUserStorage creates a new NATS-based user storage
func newNATSUserStorage(ctx context.Context, natsClient *nats.NATSClient) (internalStorageReaderWriter, error) {
	// Get the KV store for authelia users
	kvStore, exists := natsClient.GetKVStore(constants.KVBucketNameAutheliaUsers)
	if !exists {
		return nil, errs.NewUnexpected("authelia users KV bucket not found in NATS client")
	}

	slog.DebugContext(ctx, "created NATS user storage", "kvStore", kvStore)

	return &natsUserStorage{
		natsClient: natsClient,
		kvStore:    kvStore,
	}, nil
}
