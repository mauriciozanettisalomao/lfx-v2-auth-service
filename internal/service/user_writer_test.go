// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package service

import (
	"context"
	"testing"

	"github.com/linuxfoundation/lfx-v2-auth-service/internal/domain/model"
	"github.com/linuxfoundation/lfx-v2-auth-service/pkg/converters"
	"github.com/linuxfoundation/lfx-v2-auth-service/pkg/errors"
)

// mockUserWriter is a mock implementation of port.UserWriter for testing
type mockUserWriter struct {
	updateUserFunc func(ctx context.Context, user *model.User) (*model.User, error)
}

func (m *mockUserWriter) UpdateUser(ctx context.Context, user *model.User) (*model.User, error) {
	if m.updateUserFunc != nil {
		return m.updateUserFunc(ctx, user)
	}
	// Default implementation returns the user as-is
	return user, nil
}

func TestUserWriterOrchestrator_UpdateUser(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name           string
		inputUser      *model.User
		mockFunc       func(ctx context.Context, user *model.User) (*model.User, error)
		expectedUser   *model.User
		expectError    bool
		errorType      string
		validateFields func(t *testing.T, user *model.User) // Custom validation function
	}{
		{
			name: "successful update with valid user",
			inputUser: &model.User{
				Token:        "valid-token",
				Username:     "valid-username",
				UserID:       "user-123",
				PrimaryEmail: "user@example.com",
				UserMetadata: &model.UserMetadata{
					Name:     converters.StringPtr("John Doe"),
					JobTitle: converters.StringPtr("Engineer"),
				},
			},
			mockFunc: func(ctx context.Context, user *model.User) (*model.User, error) {
				// Mock successful update
				return user, nil
			},
			expectedUser: &model.User{
				Token:        "valid-token",
				Username:     "valid-username",
				UserID:       "user-123",
				PrimaryEmail: "user@example.com",
				UserMetadata: &model.UserMetadata{
					Name:     converters.StringPtr("John Doe"),
					JobTitle: converters.StringPtr("Engineer"),
				},
			},
			expectError: false,
		},
		{
			name: "validation failure - missing token",
			inputUser: &model.User{
				Username:     "valid-username",
				UserID:       "user-123",
				PrimaryEmail: "user@example.com",
			},
			expectError: true,
			errorType:   "validation",
		},
		{
			name: "validation failure - missing username",
			inputUser: &model.User{
				Token:        "valid-token",
				UserID:       "user-123",
				PrimaryEmail: "user@example.com",
			},
			expectError: true,
			errorType:   "validation",
		},
		{
			name: "sanitization and validation success",
			inputUser: &model.User{
				Token:        "  token-with-spaces  ",
				Username:     "  username-with-spaces  ",
				UserID:       "  user-123  ",
				PrimaryEmail: "  user@example.com  ",
				UserMetadata: &model.UserMetadata{
					Name:     converters.StringPtr("  John Doe  "),
					JobTitle: converters.StringPtr("  Software Engineer  "),
				},
			},
			mockFunc: func(ctx context.Context, user *model.User) (*model.User, error) {
				return user, nil
			},
			expectError: false,
			validateFields: func(t *testing.T, user *model.User) {
				// Verify sanitization occurred
				if user.Token != "token-with-spaces" {
					t.Errorf("Token not sanitized: got %q, want %q", user.Token, "token-with-spaces")
				}
				if user.Username != "username-with-spaces" {
					t.Errorf("Username not sanitized: got %q, want %q", user.Username, "username-with-spaces")
				}
				if user.UserID != "user-123" {
					t.Errorf("UserID not sanitized: got %q, want %q", user.UserID, "user-123")
				}
				if user.PrimaryEmail != "user@example.com" {
					t.Errorf("PrimaryEmail not sanitized: got %q, want %q", user.PrimaryEmail, "user@example.com")
				}
				if user.UserMetadata.Name == nil || *user.UserMetadata.Name != "John Doe" {
					t.Errorf("UserMetadata.Name not sanitized correctly")
				}
				if user.UserMetadata.JobTitle == nil || *user.UserMetadata.JobTitle != "Software Engineer" {
					t.Errorf("UserMetadata.JobTitle not sanitized correctly")
				}
			},
		},
		{
			name: "underlying service error",
			inputUser: &model.User{
				Token:        "valid-token",
				Username:     "valid-username",
				UserID:       "user-123",
				PrimaryEmail: "user@example.com",
				UserMetadata: &model.UserMetadata{
					Name: converters.StringPtr("John Doe"),
				},
			},
			mockFunc: func(ctx context.Context, user *model.User) (*model.User, error) {
				return nil, errors.NewUnexpected("database connection failed", nil)
			},
			expectError: true,
			errorType:   "unexpected",
		},
		{
			name: "validation failure - missing user_metadata",
			inputUser: &model.User{
				Token:        "  valid-token  ",
				Username:     "  valid-username  ",
				UserID:       "  user-123  ",
				PrimaryEmail: "  user@example.com  ",
				UserMetadata: nil,
			},
			expectError: true,
			errorType:   "validation",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock user writer
			mockWriter := &mockUserWriter{
				updateUserFunc: tt.mockFunc,
			}

			// Create orchestrator with mock
			orchestrator := NewUserWriterOrchestrator(
				WithUserWriter(mockWriter),
			)

			// Execute the test
			result, err := orchestrator.UpdateUser(ctx, tt.inputUser)

			// Check error expectations
			if tt.expectError {
				if err == nil {
					t.Errorf("UpdateUser() expected error, got nil")
					return
				}

				// Check error type if specified
				if tt.errorType != "" {
					switch tt.errorType {
					case "validation":
						if _, ok := err.(errors.Validation); !ok {
							t.Errorf("UpdateUser() expected Validation error, got %T: %v", err, err)
						}
					case "unexpected":
						if _, ok := err.(errors.Unexpected); !ok {
							t.Errorf("UpdateUser() expected Unexpected error, got %T: %v", err, err)
						}
					}
				}
				return
			}

			// Check success case
			if err != nil {
				t.Errorf("UpdateUser() unexpected error: %v", err)
				return
			}

			if result == nil {
				t.Errorf("UpdateUser() returned nil result")
				return
			}

			// Run custom validation if provided
			if tt.validateFields != nil {
				tt.validateFields(t, result)
			} else if tt.expectedUser != nil {
				// Basic field comparison
				if result.Token != tt.expectedUser.Token {
					t.Errorf("Token = %q, want %q", result.Token, tt.expectedUser.Token)
				}
				if result.Username != tt.expectedUser.Username {
					t.Errorf("Username = %q, want %q", result.Username, tt.expectedUser.Username)
				}
				if result.UserID != tt.expectedUser.UserID {
					t.Errorf("UserID = %q, want %q", result.UserID, tt.expectedUser.UserID)
				}
				if result.PrimaryEmail != tt.expectedUser.PrimaryEmail {
					t.Errorf("PrimaryEmail = %q, want %q", result.PrimaryEmail, tt.expectedUser.PrimaryEmail)
				}
			}
		})
	}
}

func TestNewUserWriterOrchestrator(t *testing.T) {
	t.Run("create orchestrator with options", func(t *testing.T) {
		mockWriter := &mockUserWriter{}
		orchestrator := NewUserWriterOrchestrator(
			WithUserWriter(mockWriter),
		)

		if orchestrator == nil {
			t.Error("NewUserWriterOrchestrator() returned nil")
		}

		// Verify it implements the interface
		var _ = UserServiceWriter(orchestrator)
	})

	t.Run("create orchestrator without options", func(t *testing.T) {
		orchestrator := NewUserWriterOrchestrator()

		if orchestrator == nil {
			t.Error("NewUserWriterOrchestrator() returned nil")
		}
	})
}

func TestWithUserWriter(t *testing.T) {
	t.Run("option sets user writer", func(t *testing.T) {
		mockWriter := &mockUserWriter{}

		// Create orchestrator with option
		orchestrator := NewUserWriterOrchestrator(
			WithUserWriter(mockWriter),
		)

		// Test that the writer was set by trying to use it
		ctx := context.Background()
		user := &model.User{
			Token:        "test-token",
			Username:     "test-user",
			UserID:       "user-123",
			PrimaryEmail: "test@example.com",
			UserMetadata: &model.UserMetadata{
				Name: converters.StringPtr("John Doe"),
			},
		}

		// This should not panic if the writer was set correctly
		_, err := orchestrator.UpdateUser(ctx, user)
		if err != nil {
			t.Errorf("UpdateUser() failed with properly configured orchestrator: %v", err)
		}
	})
}

// Integration test for the full flow
func TestUserWriterOrchestrator_Integration(t *testing.T) {
	ctx := context.Background()

	t.Run("full flow with sanitization and validation", func(t *testing.T) {
		// Track what the mock receives
		var receivedUser *model.User
		mockWriter := &mockUserWriter{
			updateUserFunc: func(ctx context.Context, user *model.User) (*model.User, error) {
				receivedUser = user
				// Simulate returning updated user with additional data
				updatedUser := *user
				updatedUser.Token = "updated-" + user.Token
				return &updatedUser, nil
			},
		}

		orchestrator := NewUserWriterOrchestrator(
			WithUserWriter(mockWriter),
		)

		inputUser := &model.User{
			Token:        "  original-token  ",
			Username:     "  test-user  ",
			UserID:       "  user-123  ",
			PrimaryEmail: "  test@example.com  ",
			UserMetadata: &model.UserMetadata{
				Name:         converters.StringPtr("  John Doe  "),
				Organization: converters.StringPtr("  ACME Corp  "),
			},
		}

		result, err := orchestrator.UpdateUser(ctx, inputUser)

		// Verify no error
		if err != nil {
			t.Fatalf("UpdateUser() failed: %v", err)
		}

		// Verify the mock received sanitized data
		if receivedUser.Token != "original-token" {
			t.Errorf("Mock received unsanitized token: %q", receivedUser.Token)
		}
		if receivedUser.Username != "test-user" {
			t.Errorf("Mock received unsanitized username: %q", receivedUser.Username)
		}
		if receivedUser.UserMetadata.Name == nil || *receivedUser.UserMetadata.Name != "John Doe" {
			t.Errorf("Mock received unsanitized metadata name")
		}

		// Verify the result includes the mock's changes
		if result.Token != "updated-original-token" {
			t.Errorf("Result token = %q, want %q", result.Token, "updated-original-token")
		}
	})
}
