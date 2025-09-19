// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package auth0

import (
	"context"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/linuxfoundation/lfx-v2-auth-service/internal/domain/model"
	"github.com/linuxfoundation/lfx-v2-auth-service/pkg/httpclient"
)

func TestUserWriter_jwtVerify(t *testing.T) {
	writer := &userWriter{}
	ctx := context.Background()

	// Helper function to create a test JWT token
	createTestToken := func(claims jwt.MapClaims) string {
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		tokenString, _ := token.SignedString([]byte("test-secret"))
		return tokenString
	}

	tests := []struct {
		name      string
		user      *model.User
		wantError bool
		errorMsg  string
		setupUser func() *model.User
	}{
		{
			name: "valid token with all required claims",
			setupUser: func() *model.User {
				claims := jwt.MapClaims{
					"sub":   "auth0|mauriciozanetti",
					"exp":   time.Now().Add(time.Hour).Unix(),
					"scope": "read:profile update:current_user_metadata write:data",
					"iss":   "https://linuxfoundation-dev.auth0.com/",
				}
				return &model.User{
					Token: createTestToken(claims),
				}
			},
			wantError: false,
		},
		{
			name: "empty token",
			setupUser: func() *model.User {
				return &model.User{Token: ""}
			},
			wantError: true,
			errorMsg:  "token is required",
		},
		{
			name: "token with Bearer prefix",
			setupUser: func() *model.User {
				claims := jwt.MapClaims{
					"sub":   "auth0|testuser",
					"exp":   time.Now().Add(time.Hour).Unix(),
					"scope": "update:current_user_metadata",
				}
				return &model.User{
					Token: "Bearer " + createTestToken(claims),
				}
			},
			wantError: false,
		},
		{
			name: "missing sub claim",
			setupUser: func() *model.User {
				claims := jwt.MapClaims{
					"exp":   time.Now().Add(time.Hour).Unix(),
					"scope": "update:current_user_metadata",
				}
				return &model.User{
					Token: createTestToken(claims),
				}
			},
			wantError: true,
			errorMsg:  "missing or invalid 'sub' claim in token",
		},
		{
			name: "empty sub claim",
			setupUser: func() *model.User {
				claims := jwt.MapClaims{
					"sub":   "",
					"exp":   time.Now().Add(time.Hour).Unix(),
					"scope": "update:current_user_metadata",
				}
				return &model.User{
					Token: createTestToken(claims),
				}
			},
			wantError: true,
			errorMsg:  "missing or invalid 'sub' claim in token",
		},
		{
			name: "expired token",
			setupUser: func() *model.User {
				claims := jwt.MapClaims{
					"sub":   "auth0|testuser",
					"exp":   time.Now().Add(-time.Hour).Unix(), // Expired 1 hour ago
					"scope": "update:current_user_metadata",
				}
				return &model.User{
					Token: createTestToken(claims),
				}
			},
			wantError: true,
			errorMsg:  "token has expired",
		},
		{
			name: "missing exp claim",
			setupUser: func() *model.User {
				claims := jwt.MapClaims{
					"sub":   "auth0|testuser",
					"scope": "update:current_user_metadata",
				}
				return &model.User{
					Token: createTestToken(claims),
				}
			},
			wantError: true,
			errorMsg:  "missing 'exp' claim in token",
		},
		{
			name: "missing scope claim",
			setupUser: func() *model.User {
				claims := jwt.MapClaims{
					"sub": "auth0|testuser",
					"exp": time.Now().Add(time.Hour).Unix(),
				}
				return &model.User{
					Token: createTestToken(claims),
				}
			},
			wantError: true,
			errorMsg:  "missing 'scope' claim in token",
		},
		{
			name: "missing required scope",
			setupUser: func() *model.User {
				claims := jwt.MapClaims{
					"sub":   "auth0|testuser",
					"exp":   time.Now().Add(time.Hour).Unix(),
					"scope": "read:profile write:data", // Missing update:current_user_metadata
				}
				return &model.User{
					Token: createTestToken(claims),
				}
			},
			wantError: true,
			errorMsg:  "wrong scope, got",
		},
		{
			name: "scope with multiple values including required",
			setupUser: func() *model.User {
				claims := jwt.MapClaims{
					"sub":   "auth0|testuser",
					"exp":   time.Now().Add(time.Hour).Unix(),
					"scope": "read:profile update:current_user_metadata write:data delete:files",
				}
				return &model.User{
					Token: createTestToken(claims),
				}
			},
			wantError: false,
		},
		{
			name: "invalid JWT format",
			setupUser: func() *model.User {
				return &model.User{
					Token: "invalid.jwt.token",
				}
			},
			wantError: true,
			errorMsg:  "failed to parse JWT token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user := tt.setupUser()
			originalUserID := user.UserID

			err := writer.jwtVerify(ctx, user)

			if tt.wantError {
				if err == nil {
					t.Errorf("jwtVerify() should return error")
					return
				}
				if !containsString(err.Error(), tt.errorMsg) {
					t.Errorf("jwtVerify() error = %v, should contain %v", err.Error(), tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Errorf("jwtVerify() should not return error, got %v", err)
					return
				}

				// Check that user_id was assigned from sub claim
				if user.UserID == "" {
					t.Errorf("jwtVerify() should assign user_id from sub claim")
				}
				if user.UserID == originalUserID && originalUserID == "" {
					t.Errorf("jwtVerify() should have updated user_id from empty to a value")
				}
			}
		})
	}
}

func TestUserWriter_UpdateUser(t *testing.T) {
	ctx := context.Background()

	// Create a valid token
	createValidToken := func() string {
		claims := jwt.MapClaims{
			"sub":   "auth0|testuser",
			"exp":   time.Now().Add(time.Hour).Unix(),
			"scope": "update:current_user_metadata",
		}
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		tokenString, _ := token.SignedString([]byte("test-secret"))
		return tokenString
	}

	tests := []struct {
		name      string
		config    Config
		user      *model.User
		wantError bool
		errorMsg  string
	}{
		{
			name: "missing domain configuration",
			config: Config{
				Tenant: "test-tenant",
				Domain: "", // Missing domain
			},
			user: &model.User{
				Token:        createValidToken(),
				Username:     "testuser",
				PrimaryEmail: "test@example.com",
			},
			wantError: true,
			errorMsg:  "Auth0 configuration is incomplete",
		},
		{
			name: "valid JWT validation only (no HTTP call due to missing domain)",
			config: Config{
				Tenant: "test-tenant",
				Domain: "", // This will cause HTTP call to be skipped
			},
			user: &model.User{
				Token:        createValidToken(),
				Username:     "testuser",
				PrimaryEmail: "test@example.com",
			},
			wantError: true, // Will fail due to incomplete config
			errorMsg:  "Auth0 configuration is incomplete",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			writer := NewUserReaderWriter(httpclient.DefaultConfig(), tt.config).(*userWriter)

			updatedUser, err := writer.UpdateUser(ctx, tt.user)

			if tt.wantError {
				if err == nil {
					t.Errorf("UpdateUser() should return error")
					return
				}
				if !containsString(err.Error(), tt.errorMsg) {
					t.Errorf("UpdateUser() error = %v, should contain %v", err.Error(), tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Errorf("UpdateUser() should not return error, got %v", err)
					return
				}

				// Check that user_id was assigned from token
				if updatedUser.UserID != "auth0|testuser" {
					t.Errorf("UpdateUser() should assign user_id from token, got %v", updatedUser.UserID)
				}
			}
		})
	}
}

func TestUserWriter_UpdateUser_JWTValidation(t *testing.T) {
	ctx := context.Background()

	// Create a valid token
	createValidToken := func() string {
		claims := jwt.MapClaims{
			"sub":   "auth0|testuser",
			"exp":   time.Now().Add(time.Hour).Unix(),
			"scope": "update:current_user_metadata",
		}
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		tokenString, _ := token.SignedString([]byte("test-secret"))
		return tokenString
	}

	// Test that JWT validation works correctly by testing the jwtVerify method directly
	writer := &userWriter{}

	user := &model.User{
		Token:        createValidToken(),
		Username:     "testuser",
		PrimaryEmail: "test@example.com",
	}

	// Test JWT verification directly (this should work)
	err := writer.jwtVerify(ctx, user)
	if err != nil {
		t.Errorf("jwtVerify() should not return error, got %v", err)
		return
	}

	// Check that user_id was assigned from token
	if user.UserID != "auth0|testuser" {
		t.Errorf("jwtVerify() should assign user_id from token, got %v", user.UserID)
	}
}

// Helper function to check if a string contains a substring
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		(len(s) > len(substr) &&
			(s[:len(substr)] == substr ||
				s[len(s)-len(substr):] == substr ||
				containsSubstring(s, substr))))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
