// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package auth0

import (
	"context"
	"log/slog"

	"github.com/linuxfoundation/lfx-v2-auth-service/internal/domain/model"
	"github.com/linuxfoundation/lfx-v2-auth-service/internal/domain/port"
)

type userWriter struct{}

func (u *userWriter) GetUser(ctx context.Context, user *model.User) (*model.User, error) {

	slog.InfoContext(ctx, "getting user", "user", user)

	return user, nil
}

func (u *userWriter) UpdateUser(ctx context.Context, user *model.User) (*model.User, error) {

	slog.InfoContext(ctx, "updating user", "user", user)

	return user, nil
}

// NewUserReaderWriter creates a new UserReaderWriter
func NewUserReaderWriter() port.UserReaderWriter {
	return &userWriter{}
}
