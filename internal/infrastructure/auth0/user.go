// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package auth0

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
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
	UserMetadata *model.UserMetadata `json:"user_metadata,omitempty"`
}

type userWriter struct {
	config     Config
	httpClient *httpclient.Client
}

func (u *userWriter) jwtVerify(ctx context.Context, user *model.User) error {
	if strings.TrimSpace(user.Token) == "" {
		return fmt.Errorf("token is required")
	}

	// Remove optional Bearer prefix (case-insensitive) and trim
	tokenString := strings.TrimSpace(user.Token)
	parts := strings.Fields(user.Token)
	if len(parts) > 1 && strings.EqualFold(parts[0], "Bearer") {
		tokenString = strings.Join(parts[1:], " ")
	}

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

// APIRequest represents a generic API request configuration
type APIRequest struct {
	Method      string
	Endpoint    string
	Body        interface{}
	UserID      string
	Token       string
	Description string
}

// APIResponse represents a generic API response
type APIResponse struct {
	StatusCode int
	Body       []byte
}

// callAPI makes a generic HTTP call to Auth0 Management API
func (u *userWriter) callAPI(ctx context.Context, req APIRequest) (*APIResponse, error) {
	if u.config.Domain == "" || req.Token == "" {
		return nil, errors.NewValidation(fmt.Sprintf("Auth0 configuration is incomplete: domain=%s, token_present=%t",
			u.config.Domain, req.Token != ""))
	}

	// Build the Auth0 Management API URL
	apiURL := fmt.Sprintf("https://%s/api/v2/users/%s%s", u.config.Domain, req.UserID, req.Endpoint)

	var requestBody []byte
	var err error

	// Prepare the request body if provided
	if req.Body != nil {
		requestBody, err = json.Marshal(req.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
	}

	slog.DebugContext(ctx, "calling Auth0 Management API",
		"method", req.Method,
		"user_id", req.UserID,
		"url", apiURL,
		"description", req.Description,
		"request_body", string(requestBody))

	// Prepare headers (normalize Authorization token)
	authHeader := strings.TrimSpace(req.Token)
	lower := strings.ToLower(authHeader)
	if !strings.HasPrefix(lower, "bearer ") {
		authHeader = "Bearer " + authHeader
	}
	headers := map[string]string{
		"Authorization": authHeader,
		"Accept":        "application/json",
	}

	// Add Content-Type for requests with body
	if req.Body != nil {
		headers["Content-Type"] = "application/json"
	}

	var bodyReader io.Reader
	if requestBody != nil {
		bodyReader = bytes.NewReader(requestBody)
	}

	// Make the HTTP request
	response, err := u.httpClient.Request(ctx, req.Method, apiURL, bodyReader, headers)
	if err != nil {
		slog.ErrorContext(ctx, "Auth0 Management API request failed",
			"error", err,
			"user_id", req.UserID,
			"method", req.Method,
			"description", req.Description)
		return nil, fmt.Errorf("failed to %s: %w", req.Description, err)
	}

	apiResponse := &APIResponse{
		StatusCode: response.StatusCode,
		Body:       response.Body,
	}

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		slog.ErrorContext(ctx, "Auth0 Management API returned error",
			"status_code", response.StatusCode,
			"response_body", string(response.Body),
			"user_id", req.UserID,
			"method", req.Method,
			"description", req.Description)
		return apiResponse, fmt.Errorf("Auth0 API returned status %d: %s", response.StatusCode, string(response.Body))
	}

	slog.DebugContext(ctx, "Auth0 Management API call successful",
		"user_id", req.UserID,
		"method", req.Method,
		"status_code", response.StatusCode,
		"description", req.Description)

	return apiResponse, nil
}

func (u *userWriter) GetUser(ctx context.Context, user *model.User) (*model.User, error) {

	slog.DebugContext(ctx, "getting user", "user_id", user.UserID)

	// If we don't have a user ID, we can't fetch the user
	if user.UserID == "" {
		return nil, errors.NewValidation("user_id is required to get user")
	}

	// Call Auth0 Management API to get the user
	apiRequest := APIRequest{
		Method:      http.MethodGet,
		Endpoint:    "",  // Empty for direct user endpoint
		Body:        nil, // No body for GET request
		UserID:      user.UserID,
		Token:       user.Token,
		Description: "get user details",
	}

	response, err := u.callAPI(ctx, apiRequest)
	if err != nil {
		slog.ErrorContext(ctx, "failed to get user from Auth0", "error", err, "user_id", user.UserID)
		return nil, fmt.Errorf("failed to get user from Auth0: %w", err)
	}

	// Parse the response to update the user object
	var auth0User struct {
		UserID       string              `json:"user_id"`
		Username     string              `json:"username,omitempty"`
		Email        string              `json:"email,omitempty"`
		UserMetadata *model.UserMetadata `json:"user_metadata,omitempty"`
	}

	if err := json.Unmarshal(response.Body, &auth0User); err != nil {
		slog.ErrorContext(ctx, "failed to parse Auth0 user response", "error", err, "user_id", user.UserID)
		return nil, fmt.Errorf("failed to parse user data: %w", err)
	}

	// Update the user object with data from Auth0
	if auth0User.Username != "" {
		user.Username = auth0User.Username
	}
	if auth0User.Email != "" {
		user.PrimaryEmail = auth0User.Email
	}
	if auth0User.UserMetadata != nil {
		user.UserMetadata = auth0User.UserMetadata
	}

	slog.DebugContext(ctx, "user retrieved successfully", "user_id", user.UserID)
	return user, nil
}

func (u *userWriter) UpdateUser(ctx context.Context, user *model.User) (*model.User, error) {

	if err := u.jwtVerify(ctx, user); err != nil {
		slog.ErrorContext(ctx, "jwt verify failed", "error", err)
		return nil, err
	}

	// Prepare the request body for updating user metadata
	if user.UserMetadata == nil {
		return nil, errors.NewValidation("user_metadata is required for update")
	}
	updateRequest := userUpdateRequest{UserMetadata: user.UserMetadata}

	// Call Auth0 Management API to update the user
	apiRequest := APIRequest{
		Method:      http.MethodPatch,
		Endpoint:    "", // Empty for direct user endpoint
		Body:        updateRequest,
		UserID:      user.UserID,
		Token:       user.Token,
		Description: "update user metadata",
	}

	response, err := u.callAPI(ctx, apiRequest)
	if err != nil {
		slog.ErrorContext(ctx, "failed to update user in Auth0", "error", err, "user_id", user.UserID)
		return nil, fmt.Errorf("failed to update user in Auth0: %w", err)
	}

	// Parse the Auth0 response to get the updated user metadata
	var auth0Response struct {
		UserMetadata *model.UserMetadata `json:"user_metadata,omitempty"`
	}

	if err := json.Unmarshal(response.Body, &auth0Response); err != nil {
		slog.ErrorContext(ctx, "failed to parse Auth0 update response", "error", err, "user_id", user.UserID)
		return nil, fmt.Errorf("failed to parse update response: %w", err)
	}

	// Create a new user object with only the user_metadata populated
	updatedUser := &model.User{
		UserMetadata: auth0Response.UserMetadata,
	}

	slog.InfoContext(ctx, "user updated successfully", "user_id", user.UserID)
	return updatedUser, nil
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
