// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package nats

import (
	"context"

	"github.com/linuxfoundation/lfx-v2-auth-service/internal/domain/model"
	"github.com/linuxfoundation/lfx-v2-auth-service/internal/domain/port"
	"github.com/linuxfoundation/lfx-v2-auth-service/pkg/errors"
)

type UserReaderWriter struct {
	natsClient *NATSClient
}

func (u *UserReaderWriter) GetUser(ctx context.Context, user *model.User) (*model.User, error) {
	return nil, errors.NewUnexpected("not implemented")
}

func (u *UserReaderWriter) SearchUser(ctx context.Context, user *model.User, criteria string) (*model.User, error) {
	return nil, errors.NewUnexpected("not implemented")
}

func (u *UserReaderWriter) UpdateUser(ctx context.Context, user *model.User) (*model.User, error) {
	return nil, errors.NewUnexpected("not implemented")
}

func NewUserReaderWriter(ctx context.Context, natsClient *NATSClient) port.UserReaderWriter {
	return &UserReaderWriter{
		natsClient: natsClient,
	}
}
