// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package authelia

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/linuxfoundation/lfx-v2-auth-service/internal/domain/model"
	"github.com/linuxfoundation/lfx-v2-auth-service/pkg/constants"
)

// Config holds configuration for Authelia repository
type Config struct {
	// Kubernetes ConfigMap Configuration
	ConfigMapName      string
	ConfigMapNamespace string
	UsersFileKey       string

	// Kubernetes Secret Configuration (for plain text passwords)
	SecretName      string
	SecretNamespace string

	// Kubernetes DaemonSet Configuration (for restarting Authelia)
	DaemonSetName      string
	DaemonSetNamespace string
}

// AutheliaUsersDatabase represents the structure of users_database.yml
type AutheliaUsersDatabase struct {
	Users map[string]AutheliaConfigUser `yaml:"users"`
}

// AutheliaConfigUser represents the user structure in the ConfigMap YAML
// This maintains the original Authelia format: displayname, password, email, and optional user_metadata
type AutheliaConfigUser struct {
	DisplayName  string            `yaml:"displayname"`
	Password     string            `yaml:"password"`
	Email        string            `yaml:"email"` // Remove omitempty to always include email field
	UserMetadata map[string]string `yaml:"user_metadata,omitempty"`
}

// AutheliaUser wraps model.User with Authelia-specific fields
type AutheliaUser struct {
	*model.User

	// Authelia-specific fields
	Password     string            `json:"password"`      // bcrypt hash for Authelia
	DisplayName  string            `json:"display_name"`  // display name for Authelia
	UserMetadata map[string]string `json:"user_metadata"` // Authelia metadata map
	CreatedAt    time.Time         `json:"created_at"`    // creation timestamp
	UpdatedAt    time.Time         `json:"updated_at"`    // update timestamp

	// not part of the user model, but used to track if the user is missing from the orchestrator
	// or if the password needs to be updated
	actionNeeded string
}

// SetActionNeeded sets the actionNeeded flag
func (au *AutheliaUser) SetActionNeeded(actionNeeded string) {
	au.actionNeeded = actionNeeded
}

// GetActionNeeded returns the actionNeeded flag
func (au *AutheliaUser) ActionNeeded() string {
	return au.actionNeeded
}

// ToModelUser converts AutheliaUser back to model.User
func (au *AutheliaUser) ToModelUser() *model.User {
	if au.User == nil {
		au.User = &model.User{}
	}

	// Ensure UserMetadata exists
	if au.User.UserMetadata == nil {
		au.User.UserMetadata = &model.UserMetadata{}
	}

	// Update display name
	if au.DisplayName != "" {
		au.User.UserMetadata.Name = &au.DisplayName
	}

	// Convert metadata map back to structured fields using the same names from user model JSON tags
	if au.UserMetadata != nil {
		if picture, exists := au.UserMetadata["picture"]; exists {
			au.User.UserMetadata.Picture = &picture
		}
		if zoneinfo, exists := au.UserMetadata["zoneinfo"]; exists {
			au.User.UserMetadata.Zoneinfo = &zoneinfo
		}
		if name, exists := au.UserMetadata["name"]; exists {
			au.User.UserMetadata.Name = &name
		}
		if givenName, exists := au.UserMetadata["given_name"]; exists {
			au.User.UserMetadata.GivenName = &givenName
		}
		if familyName, exists := au.UserMetadata["family_name"]; exists {
			au.User.UserMetadata.FamilyName = &familyName
		}
		if jobTitle, exists := au.UserMetadata["job_title"]; exists {
			au.User.UserMetadata.JobTitle = &jobTitle
		}
		if org, exists := au.UserMetadata["organization"]; exists {
			au.User.UserMetadata.Organization = &org
		}
		if country, exists := au.UserMetadata["country"]; exists {
			au.User.UserMetadata.Country = &country
		}
		if stateProvince, exists := au.UserMetadata["state_province"]; exists {
			au.User.UserMetadata.StateProvince = &stateProvince
		}
		if city, exists := au.UserMetadata["city"]; exists {
			au.User.UserMetadata.City = &city
		}
		if address, exists := au.UserMetadata["address"]; exists {
			au.User.UserMetadata.Address = &address
		}
		if postalCode, exists := au.UserMetadata["postal_code"]; exists {
			au.User.UserMetadata.PostalCode = &postalCode
		}
		if phoneNumber, exists := au.UserMetadata["phone_number"]; exists {
			au.User.UserMetadata.PhoneNumber = &phoneNumber
		}
		if tShirtSize, exists := au.UserMetadata["t_shirt_size"]; exists {
			au.User.UserMetadata.TShirtSize = &tShirtSize
		}
	}

	return au.User
}

// newAutheliaUser creates a new AutheliaUser from a model.User
func newAutheliaUser(user *model.User) *AutheliaUser {
	if user == nil {
		return nil
	}

	autheliaUser := &AutheliaUser{
		User:         user,
		UserMetadata: make(map[string]string),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	// Extract display name
	if user.UserMetadata != nil && user.UserMetadata.Name != nil {
		autheliaUser.DisplayName = *user.UserMetadata.Name
	}

	// Extract metadata fields using the same names from the user model JSON tags
	if user.UserMetadata != nil {
		if user.UserMetadata.Picture != nil {
			autheliaUser.UserMetadata["picture"] = *user.UserMetadata.Picture
		}
		if user.UserMetadata.Zoneinfo != nil {
			autheliaUser.UserMetadata["zoneinfo"] = *user.UserMetadata.Zoneinfo
		}
		if user.UserMetadata.Name != nil {
			autheliaUser.UserMetadata["name"] = *user.UserMetadata.Name
		}
		if user.UserMetadata.GivenName != nil {
			autheliaUser.UserMetadata["given_name"] = *user.UserMetadata.GivenName
		}
		if user.UserMetadata.FamilyName != nil {
			autheliaUser.UserMetadata["family_name"] = *user.UserMetadata.FamilyName
		}
		if user.UserMetadata.JobTitle != nil {
			autheliaUser.UserMetadata["job_title"] = *user.UserMetadata.JobTitle
		}
		if user.UserMetadata.Organization != nil {
			autheliaUser.UserMetadata["organization"] = *user.UserMetadata.Organization
		}
		if user.UserMetadata.Country != nil {
			autheliaUser.UserMetadata["country"] = *user.UserMetadata.Country
		}
		if user.UserMetadata.StateProvince != nil {
			autheliaUser.UserMetadata["state_province"] = *user.UserMetadata.StateProvince
		}
		if user.UserMetadata.City != nil {
			autheliaUser.UserMetadata["city"] = *user.UserMetadata.City
		}
		if user.UserMetadata.Address != nil {
			autheliaUser.UserMetadata["address"] = *user.UserMetadata.Address
		}
		if user.UserMetadata.PostalCode != nil {
			autheliaUser.UserMetadata["postal_code"] = *user.UserMetadata.PostalCode
		}
		if user.UserMetadata.PhoneNumber != nil {
			autheliaUser.UserMetadata["phone_number"] = *user.UserMetadata.PhoneNumber
		}
		if user.UserMetadata.TShirtSize != nil {
			autheliaUser.UserMetadata["t_shirt_size"] = *user.UserMetadata.TShirtSize
		}
	}

	return autheliaUser
}

// IsEnabled checks if Authelia is enabled
// This should be used on the flow to determine if specific functionality should be enabled,
// such as NATS KV auhtelia user storage
func IsEnabled(ctx context.Context) bool {
	enabled := os.Getenv(constants.UserRepositoryTypeEnvKey) == constants.UserRepositoryTypeAuthelia
	slog.DebugContext(ctx, "checking if Authelia is enabled",
		"user_repository_type", os.Getenv(constants.UserRepositoryTypeEnvKey),
		"enabled", enabled,
	)
	return enabled
}
