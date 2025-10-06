// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package mock

import (
	"context"
	"testing"

	"github.com/golang-jwt/jwt/v5"
)

// TestUserReaderWriter_MetadataLookup tests the MetadataLookup method for Mock implementation
func TestUserReaderWriter_MetadataLookup(t *testing.T) {
	ctx := context.Background()
	writer := &userWriter{}

	tests := []struct {
		name                 string
		input                string
		expectedSub          string
		expectedUserID       string
		expectedUsername     string
		expectedPrimaryEmail string
		expectError          bool
		errorMessage         string
	}{
		// JWT test cases
		{
			name:                 "JWT token with auth0 sub",
			input:                createTestJWT(t, "auth0|123456789"),
			expectedSub:          "auth0|123456789",
			expectedUserID:       "auth0|123456789",
			expectedUsername:     "",
			expectedPrimaryEmail: "",
			expectError:          false,
		},
		{
			name:                 "JWT token with google oauth sub",
			input:                createTestJWT(t, "google-oauth2|987654321"),
			expectedSub:          "google-oauth2|987654321",
			expectedUserID:       "google-oauth2|987654321",
			expectedUsername:     "",
			expectedPrimaryEmail: "",
			expectError:          false,
		},
		{
			name:                 "JWT token with github sub",
			input:                createTestJWT(t, "github|456789123"),
			expectedSub:          "github|456789123",
			expectedUserID:       "github|456789123",
			expectedUsername:     "",
			expectedPrimaryEmail: "",
			expectError:          false,
		},
		{
			name:                 "invalid JWT token",
			input:                "eyJinvalid.jwt.token",
			expectedSub:          "",
			expectedUserID:       "",
			expectedUsername:     "eyJinvalid.jwt.token",
			expectedPrimaryEmail: "",
			expectError:          false, // Should fall back to username lookup
		},
		// Regular test cases
		{
			name:                 "canonical lookup with pipe separator - auth0 format",
			input:                "auth0|123456789",
			expectedSub:          "auth0|123456789",
			expectedUserID:       "auth0|123456789",
			expectedUsername:     "",
			expectedPrimaryEmail: "",
			expectError:          false,
		},
		{
			name:                 "canonical lookup with google oauth format",
			input:                "google-oauth2|987654321",
			expectedSub:          "google-oauth2|987654321",
			expectedUserID:       "google-oauth2|987654321",
			expectedUsername:     "",
			expectedPrimaryEmail: "",
			expectError:          false,
		},
		{
			name:                 "canonical lookup with github oauth format",
			input:                "github|456789123",
			expectedSub:          "github|456789123",
			expectedUserID:       "github|456789123",
			expectedUsername:     "",
			expectedPrimaryEmail: "",
			expectError:          false,
		},
		{
			name:                 "canonical lookup with saml enterprise format",
			input:                "samlp|enterprise|user123",
			expectedSub:          "samlp|enterprise|user123",
			expectedUserID:       "samlp|enterprise|user123",
			expectedUsername:     "",
			expectedPrimaryEmail: "",
			expectError:          false,
		},
		{
			name:                 "canonical lookup with linkedin oauth format",
			input:                "linkedin|789123456",
			expectedSub:          "linkedin|789123456",
			expectedUserID:       "linkedin|789123456",
			expectedUsername:     "",
			expectedPrimaryEmail: "",
			expectError:          false,
		},
		{
			name:                 "canonical lookup with custom provider format",
			input:                "custom-provider|user-id-12345",
			expectedSub:          "custom-provider|user-id-12345",
			expectedUserID:       "custom-provider|user-id-12345",
			expectedUsername:     "",
			expectedPrimaryEmail: "",
			expectError:          false,
		},
		{
			name:                 "search lookup with email format",
			input:                "john.doe@example.com",
			expectedSub:          "",
			expectedUserID:       "",
			expectedUsername:     "",
			expectedPrimaryEmail: "john.doe@example.com",
			expectError:          false,
		},
		{
			name:                 "search lookup with email format - complex domain",
			input:                "jane.smith@company.co.uk",
			expectedSub:          "",
			expectedUserID:       "",
			expectedUsername:     "",
			expectedPrimaryEmail: "jane.smith@company.co.uk",
			expectError:          false,
		},
		{
			name:                 "search lookup with email format - subdomain",
			input:                "developer@mail.example.org",
			expectedSub:          "",
			expectedUserID:       "",
			expectedUsername:     "",
			expectedPrimaryEmail: "developer@mail.example.org",
			expectError:          false,
		},
		{
			name:                 "search lookup with username - no email format",
			input:                "john.doe",
			expectedSub:          "",
			expectedUserID:       "",
			expectedUsername:     "john.doe",
			expectedPrimaryEmail: "",
			expectError:          false,
		},
		{
			name:                 "search lookup with username containing numbers",
			input:                "developer123",
			expectedSub:          "",
			expectedUserID:       "",
			expectedUsername:     "developer123",
			expectedPrimaryEmail: "",
			expectError:          false,
		},
		{
			name:                 "search lookup with username containing special chars",
			input:                "jane_smith-dev",
			expectedSub:          "",
			expectedUserID:       "",
			expectedUsername:     "jane_smith-dev",
			expectedPrimaryEmail: "",
			expectError:          false,
		},
		{
			name:                 "search lookup with simple username",
			input:                "testuser",
			expectedSub:          "",
			expectedUserID:       "",
			expectedUsername:     "testuser",
			expectedPrimaryEmail: "",
			expectError:          false,
		},
		{
			name:                 "empty input",
			input:                "",
			expectedSub:          "",
			expectedUserID:       "",
			expectedUsername:     "",
			expectedPrimaryEmail: "",
			expectError:          true,
			errorMessage:         "input is required",
		},
		{
			name:                 "whitespace only input",
			input:                "   ",
			expectedSub:          "",
			expectedUserID:       "",
			expectedUsername:     "",
			expectedPrimaryEmail: "",
			expectError:          true,
			errorMessage:         "input is required",
		},
		{
			name:                 "input with leading/trailing whitespace - canonical",
			input:                "  auth0|123456789  ",
			expectedSub:          "auth0|123456789",
			expectedUserID:       "auth0|123456789",
			expectedUsername:     "",
			expectedPrimaryEmail: "",
			expectError:          false,
		},
		{
			name:                 "input with leading/trailing whitespace - email",
			input:                "  john.doe@example.com  ",
			expectedSub:          "",
			expectedUserID:       "",
			expectedUsername:     "",
			expectedPrimaryEmail: "john.doe@example.com",
			expectError:          false,
		},
		{
			name:                 "input with leading/trailing whitespace - username",
			input:                "  john.doe  ",
			expectedSub:          "",
			expectedUserID:       "",
			expectedUsername:     "john.doe",
			expectedPrimaryEmail: "",
			expectError:          false,
		},
		{
			name:                 "edge case - single pipe character",
			input:                "|",
			expectedSub:          "|",
			expectedUserID:       "|",
			expectedUsername:     "",
			expectedPrimaryEmail: "",
			expectError:          false,
		},
		{
			name:                 "edge case - pipe at end",
			input:                "provider|",
			expectedSub:          "provider|",
			expectedUserID:       "provider|",
			expectedUsername:     "",
			expectedPrimaryEmail: "",
			expectError:          false,
		},
		{
			name:                 "edge case - pipe at beginning",
			input:                "|userid",
			expectedSub:          "|userid",
			expectedUserID:       "|userid",
			expectedUsername:     "",
			expectedPrimaryEmail: "",
			expectError:          false,
		},
		{
			name:                 "edge case - multiple pipes",
			input:                "provider|connection|userid",
			expectedSub:          "provider|connection|userid",
			expectedUserID:       "provider|connection|userid",
			expectedUsername:     "",
			expectedPrimaryEmail: "",
			expectError:          false,
		},
		{
			name:                 "edge case - email with pipe (should be canonical)",
			input:                "user@domain.com|provider",
			expectedSub:          "user@domain.com|provider",
			expectedUserID:       "user@domain.com|provider",
			expectedUsername:     "",
			expectedPrimaryEmail: "",
			expectError:          false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user, err := writer.MetadataLookup(ctx, tt.input)

			// Check error expectation
			if tt.expectError {
				if err == nil {
					t.Errorf("MetadataLookup() expected error but got none")
					return
				}
				if tt.errorMessage != "" && err.Error() != tt.errorMessage {
					t.Errorf("MetadataLookup() error = %q, expected %q", err.Error(), tt.errorMessage)
				}
				return
			}

			// Check no error when not expected
			if err != nil {
				t.Errorf("MetadataLookup() unexpected error: %v", err)
				return
			}

			// Check user is not nil
			if user == nil {
				t.Errorf("MetadataLookup() returned nil user")
				return
			}

			// Check user fields are set correctly
			if user.Sub != tt.expectedSub {
				t.Errorf("MetadataLookup() Sub = %q, expected %q", user.Sub, tt.expectedSub)
			}

			if user.UserID != tt.expectedUserID {
				t.Errorf("MetadataLookup() UserID = %q, expected %q", user.UserID, tt.expectedUserID)
			}

			if user.Username != tt.expectedUsername {
				t.Errorf("MetadataLookup() Username = %q, expected %q", user.Username, tt.expectedUsername)
			}

			if user.PrimaryEmail != tt.expectedPrimaryEmail {
				t.Errorf("MetadataLookup() PrimaryEmail = %q, expected %q", user.PrimaryEmail, tt.expectedPrimaryEmail)
			}
		})
	}
}

// createTestJWT creates a test JWT token with the given sub claim
func createTestJWT(t *testing.T, sub string) string {
	t.Helper()

	// Create a new token with the sub claim
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": sub,
		"iss": "test-issuer",
		"aud": "test-audience",
		"exp": 9999999999, // Far future expiration
		"iat": 1000000000, // Past issued at
	})

	// Sign the token with a test secret (for mock purposes)
	tokenString, err := token.SignedString([]byte("test-secret"))
	if err != nil {
		t.Fatalf("Failed to create test JWT: %v", err)
	}

	return tokenString
}
