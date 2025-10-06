// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package auth0

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"

	"github.com/linuxfoundation/lfx-v2-auth-service/internal/domain/model"
	"github.com/linuxfoundation/lfx-v2-auth-service/internal/domain/port"
	"github.com/linuxfoundation/lfx-v2-auth-service/pkg/constants"
	"github.com/linuxfoundation/lfx-v2-auth-service/pkg/errors"
	"github.com/linuxfoundation/lfx-v2-auth-service/pkg/httpclient"
	jwtparser "github.com/linuxfoundation/lfx-v2-auth-service/pkg/jwt"
	"github.com/linuxfoundation/lfx-v2-auth-service/pkg/redaction"
)

const (
	userMetadataRequiredScope = "update:current_user_metadata"

	usernameFilter = "Username-Password-Authentication"
)

var (
	// criteriaEndpointMapping is a map of criteria types and their corresponding API endpoints
	criteriaEndpointMapping = map[string]string{
		constants.CriteriaTypeEmail:    "users-by-email?email=%s",
		constants.CriteriaTypeUsername: `users?q=identities.user_id:%s&search_engine=v3`,
	}
)

// Config holds the configuration for Auth0 Management API
type Config struct {
	Tenant string
	Domain string
	// M2MTokenManager for machine-to-machine authentication
	M2MTokenManager *TokenManager
}

// userUpdateRequest represents the request body for updating a user in Auth0
type userUpdateRequest struct {
	UserMetadata *model.UserMetadata `json:"user_metadata,omitempty"`
}

type userReaderWriter struct {
	config     Config
	httpClient *httpclient.Client
}

func (u *userReaderWriter) jwtVerify(ctx context.Context, user *model.User) error {
	// Configure JWT parsing options
	opts := &jwtparser.ParseOptions{
		RequireExpiration: true,
		RequiredScopes:    []string{userMetadataRequiredScope},
		AllowBearerPrefix: true,
		RequireSubject:    true,
	}

	// Parse and validate the JWT token
	claims, err := jwtparser.ParseUnverified(ctx, user.Token, opts)
	if err != nil {
		return err
	}

	// Extract the user_id from the 'sub' claim
	user.UserID = claims.Subject

	slog.DebugContext(ctx, "JWT validation successful",
		"user_id", user.UserID,
		"expires_at", claims.ExpiresAt,
		"scope", claims.Scope)

	return nil
}

func (u *userReaderWriter) SearchUser(ctx context.Context, user *model.User, criteria string) (*model.User, error) {

	endpoint, ok := criteriaEndpointMapping[criteria]
	if !ok {
		return nil, errors.NewValidation(fmt.Sprintf("invalid criteria type: %s", criteria))
	}

	param := func(criteriaType string) []any {
		switch criteriaType {
		case constants.CriteriaTypeEmail:
			slog.DebugContext(ctx, "searching user",
				"criteria", criteria,
				"email", redaction.RedactEmail(user.PrimaryEmail),
			)
			return []any{url.QueryEscape(strings.ToLower(strings.TrimSpace(user.PrimaryEmail)))}
		case constants.CriteriaTypeUsername:
			slog.DebugContext(ctx, "searching user",
				"criteria", criteria,
				"username", redaction.Redact(user.Username),
			)
			return []any{url.QueryEscape(strings.TrimSpace(user.Username))}
		}
		return []any{}
	}

	if user.Token == "" {
		m2mToken, errGetToken := u.config.M2MTokenManager.GetToken(ctx)
		if errGetToken != nil {
			return nil, errors.NewUnexpected("failed to get M2M token", errGetToken)
		}
		user.Token = m2mToken
	}

	endpointWithParam := fmt.Sprintf(endpoint, param(criteria)...)
	url := fmt.Sprintf("https://%s/api/v2/%s", u.config.Domain, endpointWithParam)

	apiRequest := httpclient.NewAPIRequest(
		u.httpClient,
		httpclient.WithMethod(http.MethodGet),
		httpclient.WithURL(url),
		httpclient.WithToken(user.Token),
		httpclient.WithDescription("search user"),
	)

	var users []Auth0User

	statusCode, errCall := apiRequest.Call(ctx, &users)
	if errCall != nil {
		slog.ErrorContext(ctx, "failed to search user",
			"error", errCall,
			"status_code", statusCode,
		)
		return nil, errors.NewUnexpected("failed to search user", errCall)
	}

	if len(users) == 0 {
		return nil, errors.NewNotFound("user not found")
	}

	slog.DebugContext(ctx, "users found, checking if the user is the one with the correct identity",
		"criteria", criteria,
	)

	for _, userResult := range users {
		// identities.user_id:{{username}} AND identities.connection:Username-Password-Authentication
		// It doesn't work like an AND, it works like an IN clause
		// (check if it contains the username and the connection, but they might not be in  the same identity)
		// So it's necessary to check if the identity is the one we are looking for
		for _, identity := range userResult.Identities {
			if identity.Connection == usernameFilter {
				// if the search is by username, we need to check if the identity is the one we are looking for
				//
				// At this point, we know that the user is found, but the validation is to
				// make sure the username is from the Username-Password-Authentication connection

				userID, ok := identity.UserID.(string)
				if !ok {
					slog.DebugContext(ctx, "user found, but it's not the correct identity",
						"filter", usernameFilter,
						"user_id", redaction.Redact(fmt.Sprintf("%v", identity.UserID)),
					)
					continue
				}

				if criteria == constants.CriteriaTypeUsername && userID != user.Username {
					slog.DebugContext(ctx, "user found, but it's not the correct identity",
						"filter", usernameFilter,
						"user_id", redaction.Redact(userID),
					)
					// if the connection is Password-Authentication and the user is not the one we are looking for,
					// we need to return an error
					return nil, errors.NewNotFound("user not found")
				}
				user.Username = userID
				return userResult.ToUser(), nil
			}
		}
	}

	return nil, errors.NewNotFound("user not found")

}

func (u *userReaderWriter) GetUser(ctx context.Context, user *model.User) (*model.User, error) {

	slog.DebugContext(ctx, "getting user", "user_id", user.UserID)

	if user.Token == "" {
		m2mToken, errGetToken := u.config.M2MTokenManager.GetToken(ctx)
		if errGetToken != nil {
			return nil, errors.NewUnexpected("failed to get M2M token", errGetToken)
		}
		user.Token = m2mToken
	}

	// If we don't have a user ID, we can't fetch the user
	if user.UserID == "" {
		return nil, errors.NewValidation("user_id is required to get user")
	}

	// Validate configuration before making HTTP requests
	if strings.TrimSpace(u.config.Domain) == "" {
		return nil, errors.NewValidation("Auth0 domain configuration is missing")
	}

	apiRequest := httpclient.NewAPIRequest(
		u.httpClient,
		httpclient.WithMethod(http.MethodGet),
		httpclient.WithURL(fmt.Sprintf("https://%s/api/v2/users/%s", u.config.Domain, user.UserID)),
		httpclient.WithToken(user.Token),
		httpclient.WithDescription("get user details"),
	)

	// Parse the response to update the user object
	var auth0User *Auth0User
	statusCode, errCall := apiRequest.Call(ctx, &auth0User)
	if errCall != nil {
		slog.ErrorContext(ctx, "failed to get user from Auth0",
			"error", errCall,
			"status_code", statusCode,
			"user_id", user.UserID,
		)
		if statusCode == http.StatusNotFound {
			return nil, errors.NewNotFound("user not found")
		}
		return nil, errors.NewUnexpected("failed to get user from Auth0", errCall)
	}

	if auth0User == nil {
		slog.ErrorContext(ctx, "failed to get user from Auth0",
			"status_code", statusCode,
			"user_id", user.UserID,
		)
		return nil, errors.NewNotFound("user not found")
	}

	slog.DebugContext(ctx, "user retrieved successfully", "user_id", user.UserID)

	return auth0User.ToUser(), nil
}

// MetadataLookup prepares the user for metadata lookup based on the input
// Returns true if should use canonical lookup, false if should use search
func (u *userReaderWriter) MetadataLookup(ctx context.Context, input string) (*model.User, error) {
	// TODO: Implement
	// breaking change, we'll address this in a future PR
	return nil, nil
}

func (u *userReaderWriter) UpdateUser(ctx context.Context, user *model.User) (*model.User, error) {

	if err := u.jwtVerify(ctx, user); err != nil {
		slog.ErrorContext(ctx, "jwt verify failed", "error", err)
		return nil, err
	}

	// Validate configuration before making HTTP requests
	if strings.TrimSpace(u.config.Domain) == "" {
		return nil, errors.NewValidation("Auth0 domain configuration is missing")
	}

	// Prepare the request body for updating user metadata
	if user.UserMetadata == nil {
		return nil, errors.NewValidation("user_metadata is required for update")
	}
	updateRequest := userUpdateRequest{UserMetadata: user.UserMetadata}

	// Call Auth0 Management API to update the user
	apiRequest := httpclient.NewAPIRequest(
		u.httpClient,
		httpclient.WithMethod(http.MethodPatch),
		httpclient.WithURL(fmt.Sprintf("https://%s/api/v2/users/%s", u.config.Domain, user.UserID)),
		httpclient.WithToken(user.Token),
		httpclient.WithDescription("update user metadata"),
		httpclient.WithBody(updateRequest),
	)

	var auth0Response struct {
		UserMetadata *model.UserMetadata `json:"user_metadata,omitempty"`
	}

	statusCode, errCall := apiRequest.Call(ctx, &auth0Response)
	if errCall != nil {
		slog.ErrorContext(ctx, "failed to update user in Auth0",
			"error", errCall,
			"status_code", statusCode,
			"user_id", user.UserID,
		)
		return nil, errors.NewUnexpected("failed to update user in Auth0", errCall)
	}

	// Create a new user object with only the user_metadata populated
	updatedUser := &model.User{
		UserMetadata: auth0Response.UserMetadata,
	}

	slog.DebugContext(ctx, "user updated successfully",
		"user_id", user.UserID,
	)
	return updatedUser, nil
}

// NewUserReaderWriter  creates a new UserReaderWriter with the provided configuration
func NewUserReaderWriter(ctx context.Context, httpConfig httpclient.Config, auth0Config Config) (port.UserReaderWriter, error) {

	// Add M2M token manager to config
	m2mTokenManager, err := NewM2MTokenManager(ctx, auth0Config)
	if err != nil {
		return nil, fmt.Errorf("failed to create M2M token manager: %w", err)
	}

	auth0Config.M2MTokenManager = m2mTokenManager

	return &userReaderWriter{
		config:     auth0Config,
		httpClient: httpclient.NewClient(httpConfig),
	}, nil
}
