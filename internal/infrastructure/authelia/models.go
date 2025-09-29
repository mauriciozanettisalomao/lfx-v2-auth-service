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
	Password     string            `json:"password"`      // bcrypt hash for Authelia
	DisplayName  string            `json:"display_name"`  // display name for Authelia
	UserMetadata map[string]string `json:"user_metadata"` // Authelia metadata map
	CreatedAt    time.Time         `json:"created_at"`    // creation timestamp
	UpdatedAt    time.Time         `json:"updated_at"`    // update timestamp

	// not part of the user model, but used to track if the user is missing from the orchestrator
	// or if the password needs to be updated
	actionNeeded string
}
