// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package jwt

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/linuxfoundation/lfx-v2-auth-service/pkg/errors"
)

// Claims represents the parsed JWT claims with commonly used fields
type Claims struct {
	Subject   string        `json:"sub"`
	ExpiresAt *time.Time    `json:"exp,omitempty"`
	IssuedAt  *time.Time    `json:"iat,omitempty"`
	NotBefore *time.Time    `json:"nbf,omitempty"`
	Issuer    string        `json:"iss,omitempty"`
	Audience  string        `json:"aud,omitempty"`
	Scope     string        `json:"scope,omitempty"`
	Raw       jwt.MapClaims `json:"-"` // Raw claims for additional fields
}

// ParseOptions configures JWT parsing behavior
type ParseOptions struct {
	// RequireExpiration validates that the token has an 'exp' claim and is not expired
	RequireExpiration bool
	// RequiredScopes validates that the token contains all specified scopes
	RequiredScopes []string
	// AllowBearerPrefix allows tokens with "Bearer " prefix
	AllowBearerPrefix bool
	// RequireSubject validates that the token has a non-empty 'sub' claim
	RequireSubject bool
}

// DefaultParseOptions returns sensible default options
func DefaultParseOptions() *ParseOptions {
	return &ParseOptions{
		RequireExpiration: true,
		AllowBearerPrefix: true,
		RequireSubject:    true,
	}
}

// ParseUnverified parses a JWT token without signature verification and returns the claims.
// This is useful for extracting information from tokens when signature verification
// is handled elsewhere or not required (e.g., in mock/test environments).
func ParseUnverified(ctx context.Context, tokenString string, opts *ParseOptions) (*Claims, error) {
	if opts == nil {
		opts = DefaultParseOptions()
	}

	if strings.TrimSpace(tokenString) == "" {
		return nil, errors.NewValidation("token is required")
	}

	// Remove optional Bearer prefix (case-insensitive) and trim
	cleanToken := strings.TrimSpace(tokenString)
	if opts.AllowBearerPrefix {
		parts := strings.Fields(tokenString)
		if len(parts) > 1 && strings.EqualFold(parts[0], "Bearer") {
			cleanToken = strings.Join(parts[1:], " ")
		}
	}

	// Parse the token without verification
	token, _, err := new(jwt.Parser).ParseUnverified(cleanToken, jwt.MapClaims{})
	if err != nil {
		return nil, errors.NewValidation("failed to parse JWT token: %w", err)
	}

	mapClaims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errors.NewValidation("invalid token claims")
	}

	claims, err := mapClaimsToClaims(mapClaims)
	if err != nil {
		return nil, err
	}

	// Validate expiration if required
	if opts.RequireExpiration {
		if err := validateExpiration(claims); err != nil {
			return nil, err
		}
	}

	// Validate subject if required
	if opts.RequireSubject {
		if err := validateSubject(claims); err != nil {
			return nil, err
		}
	}

	// Validate required scopes if specified
	if len(opts.RequiredScopes) > 0 {
		if err := validateScopes(claims, opts.RequiredScopes); err != nil {
			return nil, err
		}
	}

	slog.DebugContext(ctx, "JWT parsed successfully",
		"subject", claims.Subject,
		"expires_at", claims.ExpiresAt,
		"scope", claims.Scope)

	return claims, nil
}

// ExtractSubject is a convenience function that extracts only the 'sub' claim from a JWT token
func ExtractSubject(ctx context.Context, tokenString string) (string, error) {
	opts := &ParseOptions{
		RequireExpiration: false,
		AllowBearerPrefix: true,
		RequireSubject:    false, // We validate manually below
	}

	claims, err := ParseUnverified(ctx, tokenString, opts)
	if err != nil {
		return "", err
	}

	if strings.TrimSpace(claims.Subject) == "" {
		return "", errors.NewValidation("missing or invalid 'sub' claim in token")
	}

	slog.DebugContext(ctx, "extracted subject from JWT", "subject", claims.Subject)
	return claims.Subject, nil
}

// mapClaimsToClaims converts jwt.MapClaims to our Claims struct
func mapClaimsToClaims(mapClaims jwt.MapClaims) (*Claims, error) {
	claims := &Claims{
		Raw: mapClaims,
	}

	// Extract subject
	if sub, ok := mapClaims["sub"].(string); ok {
		claims.Subject = sub
	}

	// Extract expiration
	if exp, ok := mapClaims["exp"]; ok {
		expTime, err := parseTimeFromClaim(exp)
		if err != nil {
			return nil, errors.NewValidation("invalid 'exp' claim format: %w", err)
		}
		claims.ExpiresAt = &expTime
	}

	// Extract issued at
	if iat, ok := mapClaims["iat"]; ok {
		iatTime, err := parseTimeFromClaim(iat)
		if err != nil {
			return nil, errors.NewValidation("invalid 'iat' claim format: %w", err)
		}
		claims.IssuedAt = &iatTime
	}

	// Extract issuer
	if iss, ok := mapClaims["iss"].(string); ok {
		claims.Issuer = iss
	}

	// Extract audience
	if aud, ok := mapClaims["aud"].(string); ok {
		claims.Audience = aud
	}

	// Extract scope
	if scope, ok := mapClaims["scope"].(string); ok {
		claims.Scope = scope
	}

	return claims, nil
}

// parseTimeFromClaim handles different numeric types for time claims
func parseTimeFromClaim(claim any) (time.Time, error) {
	switch v := claim.(type) {
	case float64:
		return time.Unix(int64(v), 0), nil
	case int64:
		return time.Unix(v, 0), nil
	case int:
		return time.Unix(int64(v), 0), nil
	default:
		return time.Time{}, fmt.Errorf("unsupported time claim type: %T", claim)
	}
}

// validateSubject checks if the token has a valid subject
func validateSubject(claims *Claims) error {
	if strings.TrimSpace(claims.Subject) == "" {
		return errors.NewValidation("missing or invalid 'sub' claim in token")
	}
	return nil
}

// validateExpiration checks if the token is expired
func validateExpiration(claims *Claims) error {
	if claims.ExpiresAt == nil {
		return errors.NewValidation("missing 'exp' claim in token")
	}

	if time.Now().After(*claims.ExpiresAt) {
		return errors.NewValidation(fmt.Sprintf("token has expired at %v", *claims.ExpiresAt))
	}

	return nil
}

// validateScopes checks if the token contains all required scopes
func validateScopes(claims *Claims, requiredScopes []string) error {
	if claims.Scope == "" {
		return errors.NewValidation("missing 'scope' claim in token")
	}

	tokenScopes := strings.Fields(claims.Scope) // Split by whitespace

	for _, requiredScope := range requiredScopes {
		if !slices.Contains(tokenScopes, requiredScope) {
			return errors.NewValidation(fmt.Sprintf("missing required scope '%s', got scopes: %s", requiredScope, claims.Scope))
		}
	}

	return nil
}

// GetClaim is a helper to extract a specific claim from the raw claims
func (c *Claims) GetClaim(key string) (interface{}, bool) {
	if c.Raw == nil {
		return nil, false
	}
	value, exists := c.Raw[key]
	return value, exists
}

// GetStringClaim is a helper to extract a string claim
func (c *Claims) GetStringClaim(key string) (string, bool) {
	value, exists := c.GetClaim(key)
	if !exists {
		return "", false
	}
	str, ok := value.(string)
	return str, ok
}

// HasScope checks if the token has a specific scope
func (c *Claims) HasScope(scope string) bool {
	if c.Scope == "" {
		return false
	}
	scopes := strings.Fields(c.Scope)
	return slices.Contains(scopes, scope)
}
