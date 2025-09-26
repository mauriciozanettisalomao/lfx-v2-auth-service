// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package mock

import (
	"context"
	_ "embed"
	"fmt"
	"log/slog"

	"github.com/linuxfoundation/lfx-v2-auth-service/internal/domain/model"
	"github.com/linuxfoundation/lfx-v2-auth-service/internal/domain/port"
	"gopkg.in/yaml.v3"
)

type userWriter struct {
	// In-memory storage for mock users
	users map[string]*model.User
}

//go:embed users.yaml
var usersYAML []byte

// UserData represents the structure for YAML file
type UserData struct {
	Users []model.User `yaml:"users"`
}

// loadUsersFromYAML loads users from embedded YAML file
func loadUsersFromYAML(ctx context.Context) ([]*model.User, error) {
	var userData UserData
	if err := yaml.Unmarshal(usersYAML, &userData); err != nil {
		slog.ErrorContext(ctx, "failed to unmarshal YAML users", "error", err)
		return nil, fmt.Errorf("failed to unmarshal YAML users: %w", err)
	}

	users := make([]*model.User, len(userData.Users))
	for i := range userData.Users {
		users[i] = &userData.Users[i]
	}

	slog.InfoContext(ctx, "loaded users from embedded YAML", "count", len(users))
	return users, nil
}

func (u *userWriter) GetUser(ctx context.Context, user *model.User) (*model.User, error) {
	slog.InfoContext(ctx, "mock: getting user", "user", user)

	// For mock implementation, we'll use either username or primary email as key
	key := user.Username
	if key == "" {
		key = user.PrimaryEmail
	}

	if key == "" {
		return nil, fmt.Errorf("mock: user identifier (username or primary email) is required")
	}

	// Check if user exists in mock storage
	if existingUser, exists := u.users[key]; exists {
		slog.InfoContext(ctx, "mock: user found in storage", "key", key)
		return existingUser, nil
	}

	// If not found, return the input user as default behavior
	slog.InfoContext(ctx, "mock: user not found in storage, returning input user", "key", key)
	return user, nil
}

func (u *userWriter) SearchUser(ctx context.Context, user *model.User, criteria string) (*model.User, error) {
	slog.InfoContext(ctx, "mock: searching user", "user", user, "criteria", criteria)

	// For mock implementation, we'll search by the criteria string as a key first
	if existingUser, exists := u.users[criteria]; exists {
		slog.InfoContext(ctx, "mock: user found by criteria", "criteria", criteria)
		return existingUser, nil
	}

	// If not found by criteria, fall back to GetUser behavior
	return u.GetUser(ctx, user)
}

func (u *userWriter) UpdateUser(ctx context.Context, user *model.User) (*model.User, error) {
	slog.InfoContext(ctx, "mock: updating user", "user", user)

	// For mock implementation, we'll use either username or primary email as key
	key := user.Username
	if key == "" {
		key = user.PrimaryEmail
	}

	if key == "" {
		return nil, fmt.Errorf("mock: user identifier (username or primary email) is required")
	}

	// Get existing user from storage
	existingUser, exists := u.users[key]
	if !exists {
		// If user doesn't exist, create a new one with the provided data
		u.users[key] = user
		slog.InfoContext(ctx, "mock: new user created in storage", "key", key)
		return user, nil
	}

	// PATCH-style update: only update fields that are provided (non-empty/non-nil)
	updatedUser := *existingUser // Create a copy of the existing user

	// Update basic fields only if they're provided (non-empty)
	if user.Token != "" {
		updatedUser.Token = user.Token
	}
	if user.UserID != "" {
		updatedUser.UserID = user.UserID
	}
	if user.Username != "" {
		updatedUser.Username = user.Username
	}
	if user.PrimaryEmail != "" {
		updatedUser.PrimaryEmail = user.PrimaryEmail
	}

	// Update UserMetadata only if it's provided (not nil)
	if user.UserMetadata != nil {
		if updatedUser.UserMetadata == nil {
			// If existing user has no metadata, use the provided metadata
			updatedUser.UserMetadata = user.UserMetadata
		} else {
			// Partial update of metadata fields - only update non-nil fields
			if user.UserMetadata.Picture != nil {
				updatedUser.UserMetadata.Picture = user.UserMetadata.Picture
			}
			if user.UserMetadata.Zoneinfo != nil {
				updatedUser.UserMetadata.Zoneinfo = user.UserMetadata.Zoneinfo
			}
			if user.UserMetadata.Name != nil {
				updatedUser.UserMetadata.Name = user.UserMetadata.Name
			}
			if user.UserMetadata.GivenName != nil {
				updatedUser.UserMetadata.GivenName = user.UserMetadata.GivenName
			}
			if user.UserMetadata.FamilyName != nil {
				updatedUser.UserMetadata.FamilyName = user.UserMetadata.FamilyName
			}
			if user.UserMetadata.JobTitle != nil {
				updatedUser.UserMetadata.JobTitle = user.UserMetadata.JobTitle
			}
			if user.UserMetadata.Organization != nil {
				updatedUser.UserMetadata.Organization = user.UserMetadata.Organization
			}
			if user.UserMetadata.Country != nil {
				updatedUser.UserMetadata.Country = user.UserMetadata.Country
			}
			if user.UserMetadata.StateProvince != nil {
				updatedUser.UserMetadata.StateProvince = user.UserMetadata.StateProvince
			}
			if user.UserMetadata.City != nil {
				updatedUser.UserMetadata.City = user.UserMetadata.City
			}
			if user.UserMetadata.Address != nil {
				updatedUser.UserMetadata.Address = user.UserMetadata.Address
			}
			if user.UserMetadata.PostalCode != nil {
				updatedUser.UserMetadata.PostalCode = user.UserMetadata.PostalCode
			}
			if user.UserMetadata.PhoneNumber != nil {
				updatedUser.UserMetadata.PhoneNumber = user.UserMetadata.PhoneNumber
			}
			if user.UserMetadata.TShirtSize != nil {
				updatedUser.UserMetadata.TShirtSize = user.UserMetadata.TShirtSize
			}
		}
	}

	// Store the updated user back to storage
	u.users[key] = &updatedUser
	slog.InfoContext(ctx, "mock: user updated in storage with PATCH semantics", "key", key)

	return &updatedUser, nil
}

// NewUserReaderWriter creates a new mock UserReaderWriter with YAML file as the data source
func NewUserReaderWriter(ctx context.Context) port.UserReaderWriter {
	users := make(map[string]*model.User)

	// Load users from embedded YAML file
	mockUsers, err := loadUsersFromYAML(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "failed to load users from YAML file", "error", err)
		return &userWriter{users: users} // Return empty store if YAML fails
	}

	if len(mockUsers) == 0 {
		slog.WarnContext(ctx, "no users found in YAML file")
		return &userWriter{users: users} // Return empty store if no users
	}

	slog.InfoContext(ctx, "successfully loaded users from YAML file", "count", len(mockUsers))

	// Add users to storage with both username and email as keys for lookup flexibility
	for _, user := range mockUsers {
		users[user.Username] = user
		users[user.PrimaryEmail] = user

		slog.InfoContext(ctx, "mock: loaded user",
			"username", user.Username,
			"primary_email", user.PrimaryEmail,
			"name", func() string {
				if user.UserMetadata != nil && user.UserMetadata.Name != nil {
					return *user.UserMetadata.Name
				}
				return "unknown"
			}(),
		)
	}

	slog.InfoContext(ctx, "mock: initialized user store", "total_users", len(mockUsers), "total_keys", len(users))

	return &userWriter{
		users: users,
	}
}
