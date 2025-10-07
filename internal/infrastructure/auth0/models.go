// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package auth0

import (
	"encoding/json"
	"log/slog"

	"github.com/linuxfoundation/lfx-v2-auth-service/internal/domain/model"
)

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
	UserID     any    `json:"user_id"`
	Provider   string `json:"provider"`
	IsSocial   bool   `json:"isSocial"`
}

// Auth0UserMetadata represents the metadata of a user in Auth0.
//
// It's the same as the domain User.UserMetadata, but it might be useful
// to have it separated for Auth0 to handle it separately if needed.
type Auth0UserMetadata struct {
	Name          *string `json:"name"`
	FamilyName    *string `json:"family_name"`
	GivenName     *string `json:"given_name"`
	Picture       *string `json:"picture"`
	JobTitle      *string `json:"job_title"`
	Organization  *string `json:"organization"`
	Country       *string `json:"country"`
	StateProvince *string `json:"state_province"`
	City          *string `json:"city"`
	Address       *string `json:"address"`
	PostalCode    *string `json:"postal_code"`
	PhoneNumber   *string `json:"phone_number"`
	TShirtSize    *string `json:"t_shirt_size"`
	Zoneinfo      *string `json:"zoneinfo"`
}

// ToUser converts an Auth0User to a User
func (u *Auth0User) ToUser() *model.User {
	var meta *model.UserMetadata
	if u.UserMetadata != nil {
		meta = &model.UserMetadata{
			Name:          u.UserMetadata.Name,
			FamilyName:    u.UserMetadata.FamilyName,
			GivenName:     u.UserMetadata.GivenName,
			Picture:       u.UserMetadata.Picture,
			JobTitle:      u.UserMetadata.JobTitle,
			Organization:  u.UserMetadata.Organization,
			Country:       u.UserMetadata.Country,
			StateProvince: u.UserMetadata.StateProvince,
			City:          u.UserMetadata.City,
			Address:       u.UserMetadata.Address,
			PostalCode:    u.UserMetadata.PostalCode,
			PhoneNumber:   u.UserMetadata.PhoneNumber,
			TShirtSize:    u.UserMetadata.TShirtSize,
			Zoneinfo:      u.UserMetadata.Zoneinfo,
		}
	}
	return &model.User{
		UserID:       u.UserID,
		Username:     u.Username,
		PrimaryEmail: u.Email,
		UserMetadata: meta,
	}
}

// ErrorResponse represents an error response from Auth0
type ErrorResponse struct {
	StatusCode int    `json:"statusCode"`
	Error      string `json:"error"`
	Message    string `json:"message"`
	Attributes struct {
		Error string `json:"error"`
	} `json:"attributes"`
}

// Message returns the error message from the attributes
func (e *ErrorResponse) ErrorMessage(errorMessage string) string {
	// parse the error message from the attributes
	err := json.Unmarshal([]byte(errorMessage), e)
	if err != nil {
		slog.Error("failed to parse error message from attributes", "error", err)
		return errorMessage
	}
	return e.Message
}

// NewErrorResponse creates a new ErrorResponse
func NewErrorResponse() *ErrorResponse {
	return &ErrorResponse{}
}
