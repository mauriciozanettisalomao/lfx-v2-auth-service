// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package auth0

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"strings"

	"github.com/linuxfoundation/lfx-v2-auth-service/internal/domain/model"
	"github.com/linuxfoundation/lfx-v2-auth-service/pkg/constants"
	"github.com/linuxfoundation/lfx-v2-auth-service/pkg/errors"
	"github.com/linuxfoundation/lfx-v2-auth-service/pkg/redaction"
)

const (
	usernamePasswordAuthenticationFilter = "Username-Password-Authentication"
	emailAuthenticationFilter            = "email"
)

func filterUserByUsername(ctx context.Context, auth0User *Auth0User, user *model.User) (bool, error) {
	for _, identity := range auth0User.Identities {
		if identity.Connection == usernamePasswordAuthenticationFilter {
			// if the search is by username, we need to check if the identity is the one we are looking for
			//
			// At this point, we know that the user is found, but the validation is to
			// make sure the username is from the Username-Password-Authentication connection
			userID, ok := identity.UserID.(string)
			if !ok {
				slog.DebugContext(ctx, "user found, but it's not the correct identity",
					"filter", usernamePasswordAuthenticationFilter,
					"user_id", redaction.Redact(fmt.Sprintf("%v", identity.UserID)),
				)
				return false, nil
			}

			if userID != user.Username {
				slog.DebugContext(ctx, "user found, but it's not the correct identity",
					"filter", usernamePasswordAuthenticationFilter,
					"user_id", redaction.Redact(userID),
				)
				// if the connection is Password-Authentication and the user is not the one we are looking for,
				// we need to return an error
				return false, errors.NewNotFound("user not found")
			}
			user.Username = userID
			return true, nil
		}
	}
	return false, nil
}

func filterUserByEmail(ctx context.Context, auth0User *Auth0User, user *model.User) (bool, error) {
	for _, identity := range auth0User.Identities {
		if identity.Connection == usernamePasswordAuthenticationFilter {
			// At this point, we know that the user is found, but the validation is to
			// make sure the username is from the Username-Password-Authentication connection
			userID, ok := identity.UserID.(string)
			if !ok {
				slog.DebugContext(ctx, "user found, but it's not the correct identity",
					"filter", usernamePasswordAuthenticationFilter,
					"user_id", redaction.Redact(fmt.Sprintf("%v", identity.UserID)),
				)
				return false, nil
			}
			user.Username = userID
			return true, nil
		}
	}
	return false, nil
}

func filterUserByAlternateEmail(ctx context.Context, auth0User *Auth0User, user *model.User) (bool, error) {
	for _, identity := range auth0User.Identities {
		if identity.Connection == emailAuthenticationFilter {
			for _, alternateEmail := range user.AlternateEmail {
				if alternateEmail.Email == identity.ProfileData.Email {
					slog.DebugContext(ctx, "user found, and it's the correct identity",
						"filter", emailAuthenticationFilter,
						"identity_email", redaction.RedactEmail(identity.ProfileData.Email),
						"identity_email_verified", identity.ProfileData.EmailVerified,
					)
					auth0User.AlternateEmail = append(auth0User.AlternateEmail, Auth0ProfileData{
						Email:         identity.ProfileData.Email,
						EmailVerified: identity.ProfileData.EmailVerified,
					})
					return true, nil
				}
			}
		}
	}
	return false, nil
}

// criteriaFilter returns the arguments and filter function for the criteria type
// each filter might have a different way to filter the user, so we need to return the arguments and the filter function
func criteriaFilter(ctx context.Context, criteriaType string, user *model.User) (args []any, filter func(ctx context.Context, auth0User *Auth0User, user *model.User) (bool, error)) {

	switch criteriaType {

	case constants.CriteriaTypeEmail:
		email := strings.ToLower(strings.TrimSpace(user.PrimaryEmail))
		slog.DebugContext(ctx, "searching user",
			"criteria", criteriaType,
			"email", redaction.RedactEmail(email),
		)
		return []any{url.QueryEscape(email)}, filterUserByEmail

	case constants.CriteriaTypeUsername:
		username := strings.ToLower(strings.TrimSpace(user.Username))
		slog.DebugContext(ctx, "searching user",
			"criteria", criteriaType,
			"username", redaction.Redact(username),
		)
		return []any{url.QueryEscape(username)}, filterUserByUsername

	case constants.CriteriaTypeAlternateEmail:
		if len(user.AlternateEmail) == 0 {
			return []any{}, nil
		}
		alternateEmail := strings.ToLower(strings.TrimSpace(user.AlternateEmail[0].Email))
		slog.DebugContext(ctx, "searching user",
			"criteria", criteriaType,
			"alternate_email", redaction.RedactEmail(alternateEmail),
		)
		return []any{url.QueryEscape(alternateEmail)}, filterUserByAlternateEmail
	}
	return []any{}, nil
}
