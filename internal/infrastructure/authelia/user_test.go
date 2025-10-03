// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package authelia

import (
	"context"
	"testing"

	"github.com/linuxfoundation/lfx-v2-auth-service/internal/domain/model"
	"github.com/linuxfoundation/lfx-v2-auth-service/pkg/converters"
)

func TestUserWriter_UpdateUser_MetadataPatchBehavior(t *testing.T) {
	ctx := context.Background()

	// Test that the Patch method is called and behaves correctly
	existingUser := &AutheliaUser{
		User: &model.User{
			Username: "testuser",
			UserMetadata: &model.UserMetadata{
				Name:         converters.StringPtr("John Doe"),
				JobTitle:     converters.StringPtr("Engineer"),
				Organization: converters.StringPtr("ACME Corp"),
				Country:      converters.StringPtr("USA"),
			},
		},
	}

	inputUser := &model.User{
		Username: "testuser",
		UserMetadata: &model.UserMetadata{
			Name:     converters.StringPtr("Jane Doe"), // Update
			JobTitle: nil,                              // Should not change existing
			City:     converters.StringPtr("New York"), // New field
			// Organization and Country not specified - should be preserved
		},
	}

	mockStorage := &mockStorageReaderWriter{
		users: map[string]*AutheliaUser{
			"testuser": existingUser,
		},
	}

	userWriter := &userReaderWriter{
		storage: mockStorage,
	}

	result, err := userWriter.UpdateUser(ctx, inputUser)
	if err != nil {
		t.Fatalf("UpdateUser() failed: %v", err)
	}

	if result.UserMetadata == nil {
		t.Fatal("UpdateUser() result should have UserMetadata")
	}

	// Verify updated field
	if result.UserMetadata.Name == nil || *result.UserMetadata.Name != "Jane Doe" {
		t.Error("UpdateUser() should update Name field")
	}

	// Verify preserved fields (not specified in input)
	if result.UserMetadata.JobTitle == nil || *result.UserMetadata.JobTitle != "Engineer" {
		t.Error("UpdateUser() should preserve JobTitle when not specified in input")
	}
	if result.UserMetadata.Organization == nil || *result.UserMetadata.Organization != "ACME Corp" {
		t.Error("UpdateUser() should preserve Organization when not specified in input")
	}
	if result.UserMetadata.Country == nil || *result.UserMetadata.Country != "USA" {
		t.Error("UpdateUser() should preserve Country when not specified in input")
	}

	// Verify new field
	if result.UserMetadata.City == nil || *result.UserMetadata.City != "New York" {
		t.Error("UpdateUser() should add new City field")
	}
}
