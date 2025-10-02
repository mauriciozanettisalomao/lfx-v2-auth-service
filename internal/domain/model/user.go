// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package model

import (
	"fmt"
	"strings"

	"github.com/linuxfoundation/lfx-v2-auth-service/pkg/errors"
)

// User represents a user in the system
type User struct {
	Token        string        `json:"token" yaml:"token"`
	UserID       string        `json:"user_id" yaml:"user_id"`
	Sub          string        `json:"sub,omitempty" yaml:"sub,omitempty"`
	Username     string        `json:"username" yaml:"username"`
	PrimaryEmail string        `json:"primary_email" yaml:"primary_email"`
	UserMetadata *UserMetadata `json:"user_metadata,omitempty" yaml:"user_metadata,omitempty"`
}

// UserMetadata represents the metadata of a user
type UserMetadata struct {
	Picture       *string `json:"picture,omitempty" yaml:"picture,omitempty"`
	Zoneinfo      *string `json:"zoneinfo,omitempty" yaml:"zoneinfo,omitempty"`
	Name          *string `json:"name,omitempty" yaml:"name,omitempty"`
	GivenName     *string `json:"given_name,omitempty" yaml:"given_name,omitempty"`
	FamilyName    *string `json:"family_name,omitempty" yaml:"family_name,omitempty"`
	JobTitle      *string `json:"job_title,omitempty" yaml:"job_title,omitempty"`
	Organization  *string `json:"organization,omitempty" yaml:"organization,omitempty"`
	Country       *string `json:"country,omitempty" yaml:"country,omitempty"`
	StateProvince *string `json:"state_province,omitempty" yaml:"state_province,omitempty"`
	City          *string `json:"city,omitempty" yaml:"city,omitempty"`
	Address       *string `json:"address,omitempty" yaml:"address,omitempty"`
	PostalCode    *string `json:"postal_code,omitempty" yaml:"postal_code,omitempty"`
	PhoneNumber   *string `json:"phone_number,omitempty" yaml:"phone_number,omitempty"`
	TShirtSize    *string `json:"t_shirt_size,omitempty" yaml:"t_shirt_size,omitempty"`
}

// Validate validates the user data and returns an error if validation fails
func (u *User) Validate() error {

	errRequiredMsg := func(field string) string {
		return fmt.Sprintf("%s is required", field)
	}

	if strings.TrimSpace(u.Token) == "" {
		return errors.NewValidation(errRequiredMsg("token"))
	}

	if u.UserMetadata == nil {
		return errors.NewValidation(errRequiredMsg("user_metadata"))
	}

	return nil
}

// UserSanitize sanitizes the user data by cleaning up string fields
func (u *User) UserSanitize() {
	// Sanitize basic user fields
	u.Token = strings.TrimSpace(u.Token)
	u.UserID = strings.TrimSpace(u.UserID)
	u.Sub = strings.TrimSpace(u.Sub)
	u.Username = strings.TrimSpace(u.Username)
	u.PrimaryEmail = strings.TrimSpace(u.PrimaryEmail)

	// Sanitize UserMetadata if it exists
	if u.UserMetadata != nil {
		u.UserMetadata.userMetadataSanitize()
	}

	// TODO: add more sanitization functions as needed
}

// sanitize sanitizes the user metadata by cleaning up string fields
func (um *UserMetadata) userMetadataSanitize() {
	if um.Name != nil {
		*um.Name = strings.TrimSpace(*um.Name)
	}
	if um.GivenName != nil {
		*um.GivenName = strings.TrimSpace(*um.GivenName)
	}
	if um.FamilyName != nil {
		*um.FamilyName = strings.TrimSpace(*um.FamilyName)
	}
	if um.JobTitle != nil {
		*um.JobTitle = strings.TrimSpace(*um.JobTitle)
	}
	if um.Organization != nil {
		*um.Organization = strings.TrimSpace(*um.Organization)
	}
	if um.Country != nil {
		*um.Country = strings.TrimSpace(*um.Country)
	}
	if um.StateProvince != nil {
		*um.StateProvince = strings.TrimSpace(*um.StateProvince)
	}
	if um.City != nil {
		*um.City = strings.TrimSpace(*um.City)
	}
	if um.Address != nil {
		*um.Address = strings.TrimSpace(*um.Address)
	}
	if um.PostalCode != nil {
		*um.PostalCode = strings.TrimSpace(*um.PostalCode)
	}
	if um.PhoneNumber != nil {
		*um.PhoneNumber = strings.TrimSpace(*um.PhoneNumber)
	}
	if um.TShirtSize != nil {
		*um.TShirtSize = strings.TrimSpace(*um.TShirtSize)
	}
	if um.Picture != nil {
		*um.Picture = strings.TrimSpace(*um.Picture)
	}
	if um.Zoneinfo != nil {
		*um.Zoneinfo = strings.TrimSpace(*um.Zoneinfo)
	}
}

// Patch updates the UserMetadata with the update values only if the update values are not nil
func (a *UserMetadata) Patch(update *UserMetadata) bool {

	if update == nil {
		return false
	}

	updated := false

	if update.Picture != nil {
		a.Picture = update.Picture
		updated = true
	}

	if update.Zoneinfo != nil {
		a.Zoneinfo = update.Zoneinfo
		updated = true
	}

	if update.Name != nil {
		a.Name = update.Name
		updated = true
	}

	if update.GivenName != nil {
		a.GivenName = update.GivenName
		updated = true
	}

	if update.FamilyName != nil {
		a.FamilyName = update.FamilyName
		updated = true
	}

	if update.JobTitle != nil {
		a.JobTitle = update.JobTitle
		updated = true
	}

	if update.Organization != nil {
		a.Organization = update.Organization
		updated = true
	}

	if update.Country != nil {
		a.Country = update.Country
		updated = true
	}

	if update.StateProvince != nil {
		a.StateProvince = update.StateProvince
		updated = true
	}
	if update.City != nil {
		a.City = update.City
		updated = true
	}

	if update.Address != nil {
		a.Address = update.Address
		updated = true
	}

	if update.PostalCode != nil {
		a.PostalCode = update.PostalCode
		updated = true
	}

	if update.PhoneNumber != nil {
		a.PhoneNumber = update.PhoneNumber
		updated = true
	}

	if update.TShirtSize != nil {
		a.TShirtSize = update.TShirtSize
		updated = true
	}

	return updated
}

// PrepareForMetadataLookup prepares the user for metadata lookup based on the input
// Returns true if should use canonical lookup, false if should use search
func (u *User) PrepareForMetadataLookup(input string) bool {
	input = strings.TrimSpace(input)

	if strings.Contains(input, "|") {
		// Input contains "|", use as sub for canonical lookup
		u.Sub = input
		u.UserID = input // Auth0 uses user_id for the canonical lookup
		u.Username = ""  // Clear username field
		return true
	}
	// Input doesn't contain "|", use for search query
	u.Sub = ""    // Clear sub field
	u.UserID = "" // Clear user_id field
	u.Username = input
	return false
}
