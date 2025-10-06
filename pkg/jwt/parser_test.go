// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package jwt

import (
	"context"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseUnverified(t *testing.T) {
	ctx := context.Background()

	t.Run("valid token with all claims", func(t *testing.T) {
		// Create a test token
		now := time.Now()
		exp := now.Add(time.Hour)
		iat := now.Add(-time.Minute)

		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"sub":   "user123",
			"exp":   exp.Unix(),
			"iat":   iat.Unix(),
			"iss":   "test-issuer",
			"aud":   "test-audience",
			"scope": "read write update:current_user_metadata",
		})

		tokenString, err := token.SignedString([]byte("secret"))
		require.NoError(t, err)

		claims, err := ParseUnverified(ctx, tokenString, DefaultParseOptions())
		require.NoError(t, err)

		assert.Equal(t, "user123", claims.Subject)
		assert.NotNil(t, claims.ExpiresAt)
		assert.WithinDuration(t, exp, *claims.ExpiresAt, time.Second)
		assert.NotNil(t, claims.IssuedAt)
		assert.WithinDuration(t, iat, *claims.IssuedAt, time.Second)
		assert.Equal(t, "test-issuer", claims.Issuer)
		assert.Equal(t, "test-audience", claims.Audience)
		assert.Equal(t, "read write update:current_user_metadata", claims.Scope)
	})

	t.Run("token with Bearer prefix", func(t *testing.T) {
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"sub": "user123",
			"exp": time.Now().Add(time.Hour).Unix(),
		})

		tokenString, err := token.SignedString([]byte("secret"))
		require.NoError(t, err)

		bearerToken := "Bearer " + tokenString
		claims, err := ParseUnverified(ctx, bearerToken, DefaultParseOptions())
		require.NoError(t, err)

		assert.Equal(t, "user123", claims.Subject)
	})

	t.Run("expired token", func(t *testing.T) {
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"sub": "user123",
			"exp": time.Now().Add(-time.Hour).Unix(), // Expired
		})

		tokenString, err := token.SignedString([]byte("secret"))
		require.NoError(t, err)

		_, err = ParseUnverified(ctx, tokenString, DefaultParseOptions())
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "token has expired")
	})

	t.Run("missing required scope", func(t *testing.T) {
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"sub":   "user123",
			"exp":   time.Now().Add(time.Hour).Unix(),
			"scope": "read write",
		})

		tokenString, err := token.SignedString([]byte("secret"))
		require.NoError(t, err)

		opts := &ParseOptions{
			RequireExpiration: true,
			RequiredScopes:    []string{"update:current_user_metadata"},
			AllowBearerPrefix: true,
		}

		_, err = ParseUnverified(ctx, tokenString, opts)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "missing required scope")
	})

	t.Run("valid token with required scope", func(t *testing.T) {
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"sub":   "user123",
			"exp":   time.Now().Add(time.Hour).Unix(),
			"scope": "read write update:current_user_metadata",
		})

		tokenString, err := token.SignedString([]byte("secret"))
		require.NoError(t, err)

		opts := &ParseOptions{
			RequireExpiration: true,
			RequiredScopes:    []string{"update:current_user_metadata"},
			AllowBearerPrefix: true,
		}

		claims, err := ParseUnverified(ctx, tokenString, opts)
		require.NoError(t, err)
		assert.Equal(t, "user123", claims.Subject)
		assert.True(t, claims.HasScope("update:current_user_metadata"))
		assert.True(t, claims.HasScope("read"))
		assert.False(t, claims.HasScope("admin"))
	})

	t.Run("empty token", func(t *testing.T) {
		_, err := ParseUnverified(ctx, "", DefaultParseOptions())
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "token is required")
	})

	t.Run("invalid token format", func(t *testing.T) {
		_, err := ParseUnverified(ctx, "invalid.token", DefaultParseOptions())
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse JWT token")
	})
}

func TestExtractSubject(t *testing.T) {
	ctx := context.Background()

	t.Run("valid token", func(t *testing.T) {
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"sub": "user123",
		})

		tokenString, err := token.SignedString([]byte("secret"))
		require.NoError(t, err)

		subject, err := ExtractSubject(ctx, tokenString)
		require.NoError(t, err)
		assert.Equal(t, "user123", subject)
	})

	t.Run("token with Bearer prefix", func(t *testing.T) {
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"sub": "user456",
		})

		tokenString, err := token.SignedString([]byte("secret"))
		require.NoError(t, err)

		bearerToken := "Bearer " + tokenString
		subject, err := ExtractSubject(ctx, bearerToken)
		require.NoError(t, err)
		assert.Equal(t, "user456", subject)
	})

	t.Run("missing sub claim", func(t *testing.T) {
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"iss": "test-issuer",
		})

		tokenString, err := token.SignedString([]byte("secret"))
		require.NoError(t, err)

		_, err = ExtractSubject(ctx, tokenString)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "missing or invalid 'sub' claim")
	})

	t.Run("empty sub claim", func(t *testing.T) {
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"sub": "",
		})

		tokenString, err := token.SignedString([]byte("secret"))
		require.NoError(t, err)

		_, err = ExtractSubject(ctx, tokenString)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "missing or invalid 'sub' claim")
	})
}

func TestClaimsHelpers(t *testing.T) {
	claims := &Claims{
		Subject: "user123",
		Scope:   "read write admin",
		Raw: jwt.MapClaims{
			"custom_field": "custom_value",
			"number_field": 42,
		},
	}

	t.Run("GetClaim", func(t *testing.T) {
		value, exists := claims.GetClaim("custom_field")
		assert.True(t, exists)
		assert.Equal(t, "custom_value", value)

		_, exists = claims.GetClaim("nonexistent")
		assert.False(t, exists)
	})

	t.Run("GetStringClaim", func(t *testing.T) {
		value, ok := claims.GetStringClaim("custom_field")
		assert.True(t, ok)
		assert.Equal(t, "custom_value", value)

		_, ok = claims.GetStringClaim("number_field")
		assert.False(t, ok) // Not a string

		_, ok = claims.GetStringClaim("nonexistent")
		assert.False(t, ok)
	})

	t.Run("HasScope", func(t *testing.T) {
		assert.True(t, claims.HasScope("read"))
		assert.True(t, claims.HasScope("write"))
		assert.True(t, claims.HasScope("admin"))
		assert.False(t, claims.HasScope("delete"))
	})
}

func TestParseTimeFromClaim(t *testing.T) {
	baseTime := time.Unix(1640995200, 0) // 2022-01-01 00:00:00 UTC

	tests := []struct {
		name     string
		input    interface{}
		expected time.Time
		hasError bool
	}{
		{"float64", float64(1640995200), baseTime, false},
		{"int64", int64(1640995200), baseTime, false},
		{"int", int(1640995200), baseTime, false},
		{"string", "1640995200", time.Time{}, true},
		{"nil", nil, time.Time{}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseTimeFromClaim(tt.input)
			if tt.hasError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}
