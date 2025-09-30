// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package authelia

import (
	"time"

	"github.com/linuxfoundation/lfx-v2-auth-service/internal/domain/model"
)

// AutheliaUser wraps model.User with Authelia-specific fields
type AutheliaUser struct {
	*model.User

	// Authelia-specific fields
	Email       string    `json:"email"`       // email for Authelia
	Password    string    `json:"password"`    // bcrypt hash for Authelia
	DisplayName string    `json:"displayname"` // display name for Authelia
	CreatedAt   time.Time `json:"created_at"`  // creation timestamp
	UpdatedAt   time.Time `json:"updated_at"`  // update timestamp

	// not part of the user model, but used to track if the user is missing from the orchestrator
	// or if the password needs to be updated
	actionNeeded string
}

// AutheliaUserStorage represents the storage format for Authelia users
// This struct excludes sensitive fields like token, user_id, and primary_email
type AutheliaUserStorage struct {
	Username     string              `json:"username"`
	Email        string              `json:"email"`                   // email for Authelia
	DisplayName  string              `json:"displayname"`             // display name for Authelia
	UserMetadata *model.UserMetadata `json:"user_metadata,omitempty"` // user metadata from domain model
	CreatedAt    time.Time           `json:"created_at"`              // creation timestamp
	UpdatedAt    time.Time           `json:"updated_at"`              // update timestamp
}

// SetUsername sets the username for the user
func (a *AutheliaUser) SetUsername(username string) {
	if a.User == nil {
		a.User = &model.User{}
	}
	a.User.Username = username
}

// ToStorage converts AutheliaUser to AutheliaUserStorage for storage operations
func (a *AutheliaUser) ToStorage() *AutheliaUserStorage {
	var userMetadata *model.UserMetadata
	if a.User != nil {
		userMetadata = a.User.UserMetadata
	}

	return &AutheliaUserStorage{
		Username:     a.Username,
		Email:        a.Email,
		DisplayName:  a.DisplayName,
		UserMetadata: userMetadata,
		CreatedAt:    a.CreatedAt,
		UpdatedAt:    a.UpdatedAt,
	}
}

// FromStorage converts AutheliaUserStorage to AutheliaUser
func (a *AutheliaUser) FromStorage(storage *AutheliaUserStorage) {
	if a.User == nil {
		a.User = &model.User{}
	}
	a.Username = storage.Username
	a.User.UserMetadata = storage.UserMetadata
	a.Email = storage.Email
	a.DisplayName = storage.DisplayName
	a.CreatedAt = storage.CreatedAt
	a.UpdatedAt = storage.UpdatedAt
}

// AutheliaUserYAML represents the YAML structure for Authelia users_database.yml
type AutheliaUserYAML struct {
	DisplayName string `yaml:"displayname"`
	Password    string `yaml:"password"`
	Email       string `yaml:"email"`
}

// ToAutheliaYAML converts AutheliaUser to the format expected by Authelia
func (a *AutheliaUser) ToAutheliaYAML() AutheliaUserYAML {
	return AutheliaUserYAML{
		DisplayName: a.DisplayName,
		Password:    a.Password,
		Email:       a.Email,
	}
}

// convertUsersToAutheliaFormat converts a map of AutheliaUser to Authelia YAML format
func convertUsersToAutheliaFormat(users map[string]*AutheliaUser) map[string]map[string]AutheliaUserYAML {
	result := map[string]map[string]AutheliaUserYAML{
		"users": make(map[string]AutheliaUserYAML),
	}

	for username, user := range users {
		result["users"][username] = user.ToAutheliaYAML()
	}

	return result
}
