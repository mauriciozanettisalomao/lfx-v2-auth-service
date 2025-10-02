// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package authelia

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/linuxfoundation/lfx-v2-auth-service/internal/infrastructure/nats"
	"github.com/linuxfoundation/lfx-v2-auth-service/pkg/constants"
	errs "github.com/linuxfoundation/lfx-v2-auth-service/pkg/errors"
	"github.com/nats-io/nats.go/jetstream"
)

const (
	kvLookupPrefix = "lookup"
)

type internalStorageReaderWriter interface {
	GetUser(ctx context.Context, key string) (*AutheliaUser, error)
	ListUsers(ctx context.Context) (map[string]*AutheliaUser, error)
	SetUser(ctx context.Context, user *AutheliaUser) (any, error)
	BuildLookupKey(ctx context.Context, lookupKey, key string) string
}

// natsUserStorage implements UserStorage using NATS KV store
type natsUserStorage struct {
	natsClient *nats.NATSClient
	kvStore    jetstream.KeyValue
}

func (n *natsUserStorage) lookupUser(ctx context.Context, key string) (string, error) {

	if !strings.HasPrefix(key, kvLookupPrefix) {
		return key, nil
	}

	entry, err := n.kvStore.Get(ctx, key)
	if err != nil {
		if errors.Is(err, jetstream.ErrKeyNotFound) {
			return "", errs.NewNotFound("user not found")
		}
		return "", errs.NewUnexpected("failed to get user from NATS KV", err)
	}
	return string(entry.Value()), nil
}

func (n *natsUserStorage) GetUser(ctx context.Context, key string) (*AutheliaUser, error) {

	if key == "" {
		return nil, errs.NewUnexpected("key is required")
	}

	username, errLookupUser := n.lookupUser(ctx, key)
	if errLookupUser != nil {
		return nil, errLookupUser
	}

	entry, err := n.kvStore.Get(ctx, username)
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

		// Skip lookup keys since they are not users
		if strings.HasPrefix(key, kvLookupPrefix) {
			continue
		}

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

	// user main data
	_, errPut := n.kvStore.Put(ctx, user.Username, data)
	if errPut != nil {
		return nil, errs.NewUnexpected("failed to set user in NATS KV", errPut)
	}

	// lookup keys
	if user.Email != "" {
		user.PrimaryEmail = user.Email
		//_, errPutLookup := n.setUserLookup(ctx, "email", user.BuildEmailIndexKey(ctx), user.Username)
		_, errPutLookup := n.kvStore.Put(ctx, n.BuildLookupKey(ctx, "email", user.BuildEmailIndexKey(ctx)), []byte(user.Username))
		if errPutLookup != nil {
			return nil, errs.NewUnexpected("failed to set lookup key in NATS KV", errPutLookup)
		}
	}

	return user, nil
}

// BuildLookupKey builds the lookup key for the given lookup key and key
func (n *natsUserStorage) BuildLookupKey(ctx context.Context, lookupKey, key string) string {
	prefix := fmt.Sprintf(constants.KVLookupPrefixAuthelia, lookupKey)
	return fmt.Sprintf("%s/%s", prefix, key)
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
