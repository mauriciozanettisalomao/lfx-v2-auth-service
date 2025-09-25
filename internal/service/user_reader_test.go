// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package service

import (
	"context"
	"errors"
	"testing"

	"github.com/linuxfoundation/lfx-v2-auth-service/internal/domain/model"
)

// mockUserReader is a mock implementation of port.UserReader for testing
type mockUserReader struct {
	getUserFunc    func(ctx context.Context, user *model.User) (*model.User, error)
	searchUserFunc func(ctx context.Context, user *model.User, criteria string) (*model.User, error)
}

func (m *mockUserReader) GetUser(ctx context.Context, user *model.User) (*model.User, error) {
	if m.getUserFunc != nil {
		return m.getUserFunc(ctx, user)
	}
	return user, nil
}

func (m *mockUserReader) SearchUser(ctx context.Context, user *model.User, criteria string) (*model.User, error) {
	if m.searchUserFunc != nil {
		return m.searchUserFunc(ctx, user, criteria)
	}
	return user, nil
}

func TestUserReaderOrchestrator_GetUser(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name     string
		input    *model.User
		mockFunc func(ctx context.Context, user *model.User) (*model.User, error)
		wantErr  bool
		validate func(t *testing.T, result *model.User, err error)
	}{
		{
			name: "successful get user",
			input: &model.User{
				UserID:       "test-user-id",
				Username:     "testuser",
				PrimaryEmail: "test@example.com",
			},
			mockFunc: func(ctx context.Context, user *model.User) (*model.User, error) {
				// Simulate enriching the user data
				enrichedUser := *user
				enrichedUser.Token = "enriched-token"
				return &enrichedUser, nil
			},
			wantErr: false,
			validate: func(t *testing.T, result *model.User, err error) {
				if err != nil {
					t.Errorf("GetUser() unexpected error: %v", err)
					return
				}
				if result == nil {
					t.Error("GetUser() returned nil result")
					return
				}
				if result.UserID != "test-user-id" {
					t.Errorf("GetUser() UserID = %q, want %q", result.UserID, "test-user-id")
				}
				if result.Token != "enriched-token" {
					t.Errorf("GetUser() Token = %q, want %q", result.Token, "enriched-token")
				}
			},
		},
		{
			name: "get user with error",
			input: &model.User{
				UserID: "nonexistent-user",
			},
			mockFunc: func(ctx context.Context, user *model.User) (*model.User, error) {
				return nil, errors.New("user not found")
			},
			wantErr: true,
			validate: func(t *testing.T, result *model.User, err error) {
				if err == nil {
					t.Error("GetUser() expected error but got none")
				}
				if result != nil {
					t.Error("GetUser() should return nil result on error")
				}
			},
		},
		{
			name:  "get user with nil input",
			input: nil,
			mockFunc: func(ctx context.Context, user *model.User) (*model.User, error) {
				if user == nil {
					return nil, errors.New("user cannot be nil")
				}
				return user, nil
			},
			wantErr: true,
			validate: func(t *testing.T, result *model.User, err error) {
				if err == nil {
					t.Error("GetUser() expected error for nil input")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockReader := &mockUserReader{
				getUserFunc: tt.mockFunc,
			}

			orchestrator := NewuserReaderOrchestrator(
				WithUserReader(mockReader),
			)

			result, err := orchestrator.GetUser(ctx, tt.input)

			if (err != nil) != tt.wantErr {
				t.Errorf("GetUser() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.validate != nil {
				tt.validate(t, result, err)
			}
		})
	}
}

func TestUserReaderOrchestrator_SearchUser(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name     string
		input    *model.User
		criteria string
		mockFunc func(ctx context.Context, user *model.User, criteria string) (*model.User, error)
		wantErr  bool
		validate func(t *testing.T, result *model.User, err error)
	}{
		{
			name: "successful search user",
			input: &model.User{
				PrimaryEmail: "search@example.com",
			},
			criteria: "email",
			mockFunc: func(ctx context.Context, user *model.User, criteria string) (*model.User, error) {
				// Simulate finding user by email
				foundUser := *user
				foundUser.UserID = "found-user-id"
				foundUser.Username = "founduser"
				return &foundUser, nil
			},
			wantErr: false,
			validate: func(t *testing.T, result *model.User, err error) {
				if err != nil {
					t.Errorf("SearchUser() unexpected error: %v", err)
					return
				}
				if result == nil {
					t.Error("SearchUser() returned nil result")
					return
				}
				if result.UserID != "found-user-id" {
					t.Errorf("SearchUser() UserID = %q, want %q", result.UserID, "found-user-id")
				}
				if result.Username != "founduser" {
					t.Errorf("SearchUser() Username = %q, want %q", result.Username, "founduser")
				}
			},
		},
		{
			name: "search user not found",
			input: &model.User{
				PrimaryEmail: "notfound@example.com",
			},
			criteria: "email",
			mockFunc: func(ctx context.Context, user *model.User, criteria string) (*model.User, error) {
				return nil, errors.New("user not found")
			},
			wantErr: true,
			validate: func(t *testing.T, result *model.User, err error) {
				if err == nil {
					t.Error("SearchUser() expected error but got none")
				}
				if result != nil {
					t.Error("SearchUser() should return nil result on error")
				}
			},
		},
		{
			name: "search user with empty criteria",
			input: &model.User{
				Username: "testuser",
			},
			criteria: "",
			mockFunc: func(ctx context.Context, user *model.User, criteria string) (*model.User, error) {
				if criteria == "" {
					return nil, errors.New("search criteria cannot be empty")
				}
				return user, nil
			},
			wantErr: true,
			validate: func(t *testing.T, result *model.User, err error) {
				if err == nil {
					t.Error("SearchUser() expected error for empty criteria")
				}
			},
		},
		{
			name: "search user with complex criteria",
			input: &model.User{
				Username: "complexuser",
			},
			criteria: "username:complexuser AND active:true",
			mockFunc: func(ctx context.Context, user *model.User, criteria string) (*model.User, error) {
				// Simulate complex search
				foundUser := *user
				foundUser.UserID = "complex-user-id"
				return &foundUser, nil
			},
			wantErr: false,
			validate: func(t *testing.T, result *model.User, err error) {
				if err != nil {
					t.Errorf("SearchUser() unexpected error: %v", err)
					return
				}
				if result.UserID != "complex-user-id" {
					t.Errorf("SearchUser() UserID = %q, want %q", result.UserID, "complex-user-id")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockReader := &mockUserReader{
				searchUserFunc: tt.mockFunc,
			}

			orchestrator := NewuserReaderOrchestrator(
				WithUserReader(mockReader),
			)

			result, err := orchestrator.SearchUser(ctx, tt.input, tt.criteria)

			if (err != nil) != tt.wantErr {
				t.Errorf("SearchUser() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.validate != nil {
				tt.validate(t, result, err)
			}
		})
	}
}

func TestNewuserReaderOrchestrator(t *testing.T) {
	t.Run("create orchestrator with user reader", func(t *testing.T) {
		mockReader := &mockUserReader{}

		orchestrator := NewuserReaderOrchestrator(
			WithUserReader(mockReader),
		)

		if orchestrator == nil {
			t.Error("NewuserReaderOrchestrator() returned nil")
		}

		// Verify interface implementation
		var _ = UserServiceReader(orchestrator)
	})

	t.Run("create orchestrator without options", func(t *testing.T) {
		orchestrator := NewuserReaderOrchestrator()

		if orchestrator == nil {
			t.Error("NewuserReaderOrchestrator() returned nil")
		}
	})
}

func TestWithUserReader(t *testing.T) {
	t.Run("option sets user reader", func(t *testing.T) {
		mockReader := &mockUserReader{}

		// Create orchestrator with the option
		orchestrator := NewuserReaderOrchestrator(
			WithUserReader(mockReader),
		)

		// Cast to access internal field for testing
		if uro, ok := orchestrator.(*userReaderOrchestrator); ok {
			if uro.userReader != mockReader {
				t.Error("WithUserReader() option did not set the user reader correctly")
			}
		} else {
			t.Error("NewuserReaderOrchestrator() did not return expected type")
		}
	})
}
