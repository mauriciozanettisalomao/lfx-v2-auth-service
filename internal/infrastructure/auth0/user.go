// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package auth0

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/linuxfoundation/lfx-v2-auth-service/internal/domain/model"
	"github.com/linuxfoundation/lfx-v2-auth-service/internal/domain/port"
	"github.com/linuxfoundation/lfx-v2-auth-service/pkg/errors"
	"github.com/linuxfoundation/lfx-v2-auth-service/pkg/httpclient"
)

const (
	userMetadataRequiredScope = "update:current_user_metadata"
)

// Config holds the configuration for Auth0 Management API
type Config struct {
	Tenant string
	Domain string
}

// userUpdateRequest represents the request body for updating a user in Auth0
type userUpdateRequest struct {
	UserMetadata *model.UserMetadata `json:"user_metadata"`
}

type userWriter struct {
	config     Config
	httpClient *httpclient.Client
}

func (u *userWriter) jwtVerify(ctx context.Context, user *model.User) error {
	if strings.TrimSpace(user.Token) == "" {
		return fmt.Errorf("token is required")
	}

	// Remove "Bearer " prefix if present
	tokenString := strings.TrimPrefix(user.Token, "Bearer ")
	tokenString = strings.TrimSpace(tokenString)

	// Parse the token without verification for now (we'll add JWKS verification later if needed)
	token, _, err := new(jwt.Parser).ParseUnverified(tokenString, jwt.MapClaims{})
	if err != nil {
		return errors.NewValidation("failed to parse JWT token: %w", err)
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return errors.NewValidation("invalid token claims")
	}

	// 1. Extract user_id from 'sub' claim
	sub, ok := claims["sub"].(string)
	if !ok || strings.TrimSpace(sub) == "" {
		return errors.NewValidation("missing or invalid 'sub' claim in token")
	}

	// Assign the user_id from sub claim
	user.UserID = sub

	slog.DebugContext(ctx, "extracted user_id from token", "user_id", user.UserID)

	// 2. Check if token is expired
	exp, okExp := claims["exp"]
	if !okExp {
		return errors.NewValidation("missing 'exp' claim in token")
	}
	var expTime time.Time
	switch expValue := exp.(type) {
	case float64:
		expTime = time.Unix(int64(expValue), 0)
	case int64:
		expTime = time.Unix(expValue, 0)
	case int:
		expTime = time.Unix(int64(expValue), 0)
	default:
		return errors.NewValidation("invalid 'exp' claim format")
	}
	if time.Now().After(expTime) {
		return errors.NewValidation(fmt.Sprintf("token has expired at %v", expTime))
	}
	slog.DebugContext(ctx, "token expiration validated", "expires_at", expTime)

	// 3. Check if scope contains 'update:current_user_metadata'
	scopeClaim, okScopeClaim := claims["scope"]
	if !okScopeClaim {
		return errors.NewValidation("missing 'scope' claim in token")
	}
	scopeString, ok := scopeClaim.(string)
	if !ok {
		return errors.NewValidation("invalid 'scope' claim format")
	}

	scopes := strings.Fields(scopeString) // Split by whitespace
	hasRequiredScope := slices.Contains(scopes, userMetadataRequiredScope)
	if !hasRequiredScope {
		return errors.NewValidation(fmt.Sprintf("wrong scope, got %s", scopeString))
	}

	slog.DebugContext(ctx, "JWT validation successful", "user_id", user.UserID)
	return nil
}

// callAuth0ManagementAPI makes an HTTP call to Auth0 Management API to update user metadata
func (u *userWriter) callAuth0ManagementAPI(ctx context.Context, user *model.User) error {
	if u.config.Domain == "" || user.Token == "" {
		return errors.NewValidation(fmt.Sprintf("Auth0 configuration is incomplete: domain=%s, token_present=%t",
			u.config.Domain, user.Token != ""))
	}

	// Build the Auth0 Management API URL
	apiURL := fmt.Sprintf("https://%s/api/v2/users/%s", u.config.Domain, user.UserID)

	// Prepare the request body
	updateRequest := userUpdateRequest{
		UserMetadata: user.UserMetadata,
	}

	requestBody, err := json.Marshal(updateRequest)
	if err != nil {
		return fmt.Errorf("failed to marshal update request: %w", err)
	}

	slog.DebugContext(ctx, "calling Auth0 Management API",
		"user_id", user.UserID,
		"url", apiURL,
		"request_body", string(requestBody))

	// Make the HTTP request
	headers := map[string]string{
		"Authorization": fmt.Sprintf("Bearer %s", user.Token),
		"Content-Type":  "application/json",
	}

	response, err := u.httpClient.Request(ctx, http.MethodPatch, apiURL, bytes.NewReader(requestBody), headers)
	if err != nil {
		slog.ErrorContext(ctx, "Auth0 Management API request failed", "error", err, "user_id", user.UserID)
		return fmt.Errorf("failed to update user in Auth0: %w", err)
	}

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		slog.ErrorContext(ctx, "Auth0 Management API returned error",
			"status_code", response.StatusCode,
			"response_body", string(response.Body),
			"user_id", user.UserID)
		return fmt.Errorf("Auth0 API returned status %d: %s", response.StatusCode, string(response.Body))
	}

	slog.InfoContext(ctx, "successfully updated user in Auth0",
		"user_id", user.UserID,
		"status_code", response.StatusCode)

	return nil
}

func (u *userWriter) GetUser(ctx context.Context, user *model.User) (*model.User, error) {

	slog.DebugContext(ctx, "getting user", "user", user)

	return user, nil
}

func (u *userWriter) UpdateUser(ctx context.Context, user *model.User) (*model.User, error) {

	if err := u.jwtVerify(ctx, user); err != nil {
		slog.ErrorContext(ctx, "jwt verify failed", "error", err)
		return nil, err
	}

	// Call Auth0 Management API to update the user
	if err := u.callAuth0ManagementAPI(ctx, user); err != nil {
		slog.ErrorContext(ctx, "failed to update user in Auth0", "error", err, "user_id", user.UserID)
		return nil, fmt.Errorf("failed to update user in Auth0: %w", err)
	}

	slog.InfoContext(ctx, "user updated successfully", "user_id", user.UserID)
	return user, nil
}

// NewUserReaderWriter creates a new UserReaderWriter with the provided configuration
func NewUserReaderWriter(httpConfig httpclient.Config, auth0Config Config) port.UserReaderWriter {
	// Create HTTP client with default configuration
	httpClient := httpclient.NewClient(httpConfig)

	return &userWriter{
		config:     auth0Config,
		httpClient: httpClient,
	}
}
