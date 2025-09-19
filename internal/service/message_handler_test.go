// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package service

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/linuxfoundation/lfx-v2-auth-service/internal/domain/model"
	"github.com/linuxfoundation/lfx-v2-auth-service/pkg/converters"
	"github.com/linuxfoundation/lfx-v2-auth-service/pkg/errors"
)

// mockTransportMessenger is a mock implementation of port.TransportMessenger for testing
type mockTransportMessenger struct {
	data []byte
}

func (m *mockTransportMessenger) Subject() string {
	return "test-subject"
}

func (m *mockTransportMessenger) Data() []byte {
	return m.data
}

func (m *mockTransportMessenger) Respond(data []byte) error {
	// Mock implementation - just return nil
	return nil
}

// mockUserServiceWriter is a mock implementation of UserServiceWriter for testing
type mockUserServiceWriter struct {
	updateUserFunc func(ctx context.Context, user *model.User) (*model.User, error)
}

func (m *mockUserServiceWriter) UpdateUser(ctx context.Context, user *model.User) (*model.User, error) {
	if m.updateUserFunc != nil {
		return m.updateUserFunc(ctx, user)
	}
	return user, nil
}

func TestMessageHandlerOrchestrator_UpdateUser(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name           string
		messageData    []byte
		mockFunc       func(ctx context.Context, user *model.User) (*model.User, error)
		expectError    bool
		errorType      string
		validateResult func(t *testing.T, result []byte)
	}{
		{
			name: "successful user update",
			messageData: func() []byte {
				user := &model.User{
					Token:        "test-token",
					Username:     "test-user",
					UserID:       "user-123",
					PrimaryEmail: "test@example.com",
					UserMetadata: &model.UserMetadata{
						Name:     converters.StringPtr("John Doe"),
						JobTitle: converters.StringPtr("Engineer"),
					},
				}
				data, _ := json.Marshal(user)
				return data
			}(),
			mockFunc: func(ctx context.Context, user *model.User) (*model.User, error) {
				// Simulate successful update with modifications
				updatedUser := *user
				updatedUser.Token = "updated-" + user.Token
				return &updatedUser, nil
			},
			expectError: false,
			validateResult: func(t *testing.T, result []byte) {
				var response struct {
					Success bool        `json:"success"`
					Data    interface{} `json:"data"`
					Error   string      `json:"error"`
				}
				if err := json.Unmarshal(result, &response); err != nil {
					t.Fatalf("Failed to unmarshal result: %v", err)
				}
				if !response.Success {
					t.Errorf("Expected success=true, got success=%v, error=%s", response.Success, response.Error)
				}
				if response.Data == nil {
					t.Fatal("Expected data, got nil")
				}
				// Since we're only returning metadata, we can't validate token/username anymore
				// The test should validate the metadata content instead
				if metadata, ok := response.Data.(map[string]interface{}); ok {
					if name, exists := metadata["name"]; exists && name != "John Doe" {
						t.Errorf("Expected name 'John Doe', got %v", name)
					}
				}
			},
		},
		{
			name:        "invalid JSON in message",
			messageData: []byte(`{invalid json`),
			expectError: true,
			errorType:   "unexpected",
		},
		{
			name: "empty message data",
			messageData: func() []byte {
				return []byte(`{}`)
			}(),
			mockFunc: func(ctx context.Context, user *model.User) (*model.User, error) {
				// This should fail validation due to missing required fields
				return nil, errors.NewValidation("username is required")
			},
			expectError: true,
			errorType:   "unexpected",
		},
		{
			name: "user service writer error",
			messageData: func() []byte {
				user := &model.User{
					Token:        "test-token",
					Username:     "test-user",
					UserID:       "user-123",
					PrimaryEmail: "test@example.com",
				}
				data, _ := json.Marshal(user)
				return data
			}(),
			mockFunc: func(ctx context.Context, user *model.User) (*model.User, error) {
				return nil, errors.NewUnexpected("database connection failed", nil)
			},
			expectError: true,
			errorType:   "unexpected",
		},
		{
			name: "user with minimal data",
			messageData: func() []byte {
				user := &model.User{
					Token:    "minimal-token",
					Username: "minimal-user",
				}
				data, _ := json.Marshal(user)
				return data
			}(),
			mockFunc: func(ctx context.Context, user *model.User) (*model.User, error) {
				return user, nil
			},
			expectError: false,
			validateResult: func(t *testing.T, result []byte) {
				var response struct {
					Success bool        `json:"success"`
					Data    interface{} `json:"data"`
					Error   string      `json:"error"`
				}
				if err := json.Unmarshal(result, &response); err != nil {
					t.Fatalf("Failed to unmarshal result: %v", err)
				}
				if !response.Success {
					t.Errorf("Expected success=true, got success=%v, error=%s", response.Success, response.Error)
				}
				// Should have nil metadata since no metadata was provided
				if response.Data != nil {
					t.Errorf("Expected nil data for minimal user, got %v", response.Data)
				}
			},
		},
		{
			name: "user with complete metadata",
			messageData: func() []byte {
				user := &model.User{
					Token:        "complete-token",
					Username:     "complete-user",
					UserID:       "user-456",
					PrimaryEmail: "complete@example.com",
					UserMetadata: &model.UserMetadata{
						Name:          converters.StringPtr("Jane Smith"),
						GivenName:     converters.StringPtr("Jane"),
						FamilyName:    converters.StringPtr("Smith"),
						JobTitle:      converters.StringPtr("Senior Engineer"),
						Organization:  converters.StringPtr("Tech Corp"),
						Country:       converters.StringPtr("USA"),
						StateProvince: converters.StringPtr("California"),
						City:          converters.StringPtr("San Francisco"),
						Address:       converters.StringPtr("123 Tech St"),
						PostalCode:    converters.StringPtr("94105"),
						PhoneNumber:   converters.StringPtr("+1-555-123-4567"),
						TShirtSize:    converters.StringPtr("M"),
						Picture:       converters.StringPtr("https://example.com/pic.jpg"),
						Zoneinfo:      converters.StringPtr("America/Los_Angeles"),
					},
				}
				data, _ := json.Marshal(user)
				return data
			}(),
			mockFunc: func(ctx context.Context, user *model.User) (*model.User, error) {
				return user, nil
			},
			expectError: false,
			validateResult: func(t *testing.T, result []byte) {
				var response struct {
					Success bool        `json:"success"`
					Data    interface{} `json:"data"`
					Error   string      `json:"error"`
				}
				if err := json.Unmarshal(result, &response); err != nil {
					t.Fatalf("Failed to unmarshal result: %v", err)
				}
				if !response.Success {
					t.Errorf("Expected success=true, got success=%v, error=%s", response.Success, response.Error)
				}
				if response.Data == nil {
					t.Fatal("Expected data, got nil")
				}

				// Verify metadata fields by casting to map
				if metadata, ok := response.Data.(map[string]interface{}); ok {
					if name, exists := metadata["name"]; exists && name != "Jane Smith" {
						t.Errorf("Result metadata name incorrect: got %v, want Jane Smith", name)
					}
					if jobTitle, exists := metadata["job_title"]; exists && jobTitle != "Senior Engineer" {
						t.Errorf("Result metadata job title incorrect: got %v, want Senior Engineer", jobTitle)
					}
					if organization, exists := metadata["organization"]; exists && organization != "Tech Corp" {
						t.Errorf("Result metadata organization incorrect: got %v, want Tech Corp", organization)
					}
				} else {
					t.Errorf("Data is not a map[string]interface{}, got %T", response.Data)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock transport messenger
			mockMsg := &mockTransportMessenger{
				data: tt.messageData,
			}

			// Create mock user service writer
			mockWriter := &mockUserServiceWriter{
				updateUserFunc: tt.mockFunc,
			}

			// Create orchestrator with mock
			orchestrator := NewMessageHandlerOrchestrator(
				WithUserWriterForMessageHandler(mockWriter),
			)

			// Execute the test
			result, err := orchestrator.UpdateUser(ctx, mockMsg)

			// Since we now return structured responses, we should never get Go errors
			if err != nil {
				t.Errorf("UpdateUser() unexpected error: %v", err)
				return
			}

			if result == nil {
				t.Errorf("UpdateUser() returned nil result")
				return
			}

			// Run custom validation if provided
			if tt.validateResult != nil {
				tt.validateResult(t, result)
			}
		})
	}
}

func TestNewMessageHandlerOrchestrator(t *testing.T) {
	t.Run("create orchestrator with options", func(t *testing.T) {
		mockWriter := &mockUserServiceWriter{}
		orchestrator := NewMessageHandlerOrchestrator(
			WithUserWriterForMessageHandler(mockWriter),
		)

		if orchestrator == nil {
			t.Error("NewMessageHandlerOrchestrator() returned nil")
		}
	})

	t.Run("create orchestrator without options", func(t *testing.T) {
		orchestrator := NewMessageHandlerOrchestrator()

		if orchestrator == nil {
			t.Error("NewMessageHandlerOrchestrator() returned nil")
		}
	})
}

func TestWithUserWriterForMessageHandler(t *testing.T) {
	t.Run("option sets user writer", func(t *testing.T) {
		mockWriter := &mockUserServiceWriter{
			updateUserFunc: func(ctx context.Context, user *model.User) (*model.User, error) {
				// Mark that this was called
				user.Token = "writer-called"
				return user, nil
			},
		}

		// Create orchestrator with option
		orchestrator := NewMessageHandlerOrchestrator(
			WithUserWriterForMessageHandler(mockWriter),
		)

		// Test that the writer was set by using it
		ctx := context.Background()
		user := &model.User{
			Token:        "test-token",
			Username:     "test-user",
			UserID:       "user-123",
			PrimaryEmail: "test@example.com",
		}
		userData, _ := json.Marshal(user)
		mockMsg := &mockTransportMessenger{data: userData}

		result, err := orchestrator.UpdateUser(ctx, mockMsg)
		if err != nil {
			t.Errorf("UpdateUser() failed: %v", err)
			return
		}

		// Verify we get a structured response with success=true
		var response struct {
			Success bool        `json:"success"`
			Data    interface{} `json:"data"`
			Error   string      `json:"error"`
		}
		if err := json.Unmarshal(result, &response); err != nil {
			t.Fatalf("Failed to unmarshal result: %v", err)
		}

		if !response.Success {
			t.Errorf("Expected success=true, got success=%v, error=%s", response.Success, response.Error)
		}

		// The result should have nil metadata since the test user has no metadata
		if response.Data != nil {
			t.Errorf("Expected nil user_metadata, got %v", response.Data)
		}
	})
}

// Integration test for the full message handling flow
func TestMessageHandlerOrchestrator_Integration(t *testing.T) {
	ctx := context.Background()

	t.Run("full message handling flow", func(t *testing.T) {
		// Track what the mock receives and processes
		var receivedUser *model.User
		mockWriter := &mockUserServiceWriter{
			updateUserFunc: func(ctx context.Context, user *model.User) (*model.User, error) {
				receivedUser = user
				// Simulate processing by the user writer (including validation/sanitization)
				processedUser := *user
				processedUser.Token = "processed-" + user.Token

				// Add some metadata to simulate enrichment
				if processedUser.UserMetadata == nil {
					processedUser.UserMetadata = &model.UserMetadata{}
				}
				processedUser.UserMetadata.Organization = converters.StringPtr("Processed Corp")

				return &processedUser, nil
			},
		}

		orchestrator := NewMessageHandlerOrchestrator(
			WithUserWriterForMessageHandler(mockWriter),
		)

		// Create input message
		inputUser := &model.User{
			Token:        "original-token",
			Username:     "integration-user",
			UserID:       "user-integration",
			PrimaryEmail: "integration@example.com",
			UserMetadata: &model.UserMetadata{
				Name:     converters.StringPtr("Integration Test"),
				JobTitle: converters.StringPtr("Tester"),
			},
		}
		messageData, err := json.Marshal(inputUser)
		if err != nil {
			t.Fatalf("Failed to marshal input user: %v", err)
		}

		mockMsg := &mockTransportMessenger{data: messageData}

		// Execute the integration test
		result, err := orchestrator.UpdateUser(ctx, mockMsg)
		if err != nil {
			t.Fatalf("Integration test failed: %v", err)
		}

		// Verify the mock received the correct data
		if receivedUser == nil {
			t.Fatal("Mock writer did not receive user data")
		}
		if receivedUser.Token != "original-token" {
			t.Errorf("Mock received incorrect token: %q", receivedUser.Token)
		}
		if receivedUser.Username != "integration-user" {
			t.Errorf("Mock received incorrect username: %q", receivedUser.Username)
		}

		// Verify the final result is a structured response with success=true
		var response struct {
			Success bool        `json:"success"`
			Data    interface{} `json:"data"`
			Error   string      `json:"error"`
		}
		if err := json.Unmarshal(result, &response); err != nil {
			t.Fatalf("Failed to unmarshal final result: %v", err)
		}

		if !response.Success {
			t.Errorf("Expected success=true, got success=%v, error=%s", response.Success, response.Error)
		}

		if response.Data == nil {
			t.Fatal("Expected data, got nil")
		}

		// Verify enrichment occurred by casting to map
		if metadata, ok := response.Data.(map[string]interface{}); ok {
			if organization, exists := metadata["organization"]; !exists || organization != "Processed Corp" {
				t.Errorf("User metadata was not enriched correctly, got organization: %v", organization)
			}

			// Verify original metadata was preserved
			if name, exists := metadata["name"]; !exists || name != "Integration Test" {
				t.Errorf("Original metadata was not preserved, got name: %v", name)
			}
		} else {
			t.Errorf("Data is not a map[string]interface{}, got %T", response.Data)
		}
	})

	t.Run("error handling in integration", func(t *testing.T) {
		// Mock that returns an error
		mockWriter := &mockUserServiceWriter{
			updateUserFunc: func(ctx context.Context, user *model.User) (*model.User, error) {
				return nil, errors.NewValidation("integration test error")
			},
		}

		orchestrator := NewMessageHandlerOrchestrator(
			WithUserWriterForMessageHandler(mockWriter),
		)

		inputUser := &model.User{
			Token:        "error-token",
			Username:     "error-user",
			UserID:       "user-error",
			PrimaryEmail: "error@example.com",
		}
		messageData, _ := json.Marshal(inputUser)
		mockMsg := &mockTransportMessenger{data: messageData}

		// Execute and expect structured error response
		result, err := orchestrator.UpdateUser(ctx, mockMsg)

		if err != nil {
			t.Errorf("Expected no error from orchestrator, got %v", err)
		}
		if result == nil {
			t.Fatal("Expected structured error response, got nil")
		}

		// Verify it's a structured error response
		var response struct {
			Success bool   `json:"success"`
			Error   string `json:"error"`
		}
		if err := json.Unmarshal(result, &response); err != nil {
			t.Fatalf("Failed to unmarshal error response: %v", err)
		}

		if response.Success {
			t.Error("Expected success=false for error case")
		}

		if response.Error != "integration test error" {
			t.Errorf("Expected 'integration test error', got %s", response.Error)
		}
	})
}
