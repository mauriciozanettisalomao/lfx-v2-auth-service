// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package authelia

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/linuxfoundation/lfx-v2-auth-service/internal/domain/model"
	"github.com/linuxfoundation/lfx-v2-auth-service/internal/infrastructure/nats"
	"github.com/linuxfoundation/lfx-v2-auth-service/pkg/constants"
	"github.com/linuxfoundation/lfx-v2-auth-service/pkg/errors"
	"github.com/nats-io/nats.go/jetstream"
)

type internalStorageReaderWriter interface {
	GetUser(ctx context.Context, user *model.User) (any, error)
	SearchUser(ctx context.Context, user *model.User, criteria string) (any, error)
	UpdateUser(ctx context.Context, user *model.User) (any, error)
}

// natsUserStorage implements UserStorage using NATS KV store
type natsUserStorage struct {
	natsClient *nats.NATSClient
	kvStore    jetstream.KeyValue
}

func (n *natsUserStorage) GetUser(ctx context.Context, user *model.User) (any, error) {

	if user == nil || user.Username == "" {
		return nil, errors.NewUnexpected("user is required")
	}

	entry, err := n.kvStore.Get(ctx, user.Username)
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

func (n *natsUserStorage) SearchUser(ctx context.Context, user *model.User, criteria string) (any, error) {
	return nil, errors.NewUnexpected("not implemented")
}

func (n *natsUserStorage) UpdateUser(ctx context.Context, user *model.User) (any, error) {
	return nil, errors.NewUnexpected("not implemented")
}

// newNATSUserStorage creates a new NATS-based user storage
func newNATSUserStorage(ctx context.Context, natsClient *nats.NATSClient) (internalStorageReaderWriter, error) {
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
