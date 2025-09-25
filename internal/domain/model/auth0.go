// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package model

// Auth0User represents a user in Auth0
type Auth0User struct {
	UserID        string             `json:"user_id"`
	Username      string             `json:"username"`
	Email         string             `json:"email"`
	EmailVerified bool               `json:"email_verified"`
	FamilyName    string             `json:"family_name"`
	GivenName     string             `json:"given_name"`
	Identities    []Auth0Identity    `json:"identities"`
	UserMetadata  *Auth0UserMetadata `json:"user_metadata"`
}

// Auth0Identity represents an identity in Auth0
type Auth0Identity struct {
	Connection string `json:"connection"`
	UserID     string `json:"user_id"`
	Provider   string `json:"provider"`
	IsSocial   bool   `json:"isSocial"`
}

// Auth0UserMetadata represents the metadata of a user in Auth0
type Auth0UserMetadata struct {
	Name       *string `json:"name"`
	FamilyName *string `json:"family_name"`
	GivenName  *string `json:"given_name"`
	Picture    *string `json:"picture"`
	JobTitle   *string `json:"job_title"`
}

// ToUser converts an Auth0User to a User
func (u *Auth0User) ToUser() *User {
	return &User{
		UserID:       u.UserID,
		Username:     u.Username,
		PrimaryEmail: u.Email,
		UserMetadata: &UserMetadata{
			Name:       u.UserMetadata.Name,
			FamilyName: u.UserMetadata.FamilyName,
			GivenName:  u.UserMetadata.GivenName,
			Picture:    u.UserMetadata.Picture,
			JobTitle:   u.UserMetadata.JobTitle,
		},
	}
}
