# Auth0 Integration

This package provides Auth0 integration for the LFX v2 Auth Service, implementing user management operations through the Auth0 Management API.

## Overview

The Auth0 integration takes a JWT token and validates/retrieves user data from the Auth0 identity provider. The system parses the JWT token to extract user identification information and performs lookups through the Auth0 Management API.

## Token Support

The Auth0 integration supports JWT (JSON Web Token) parsing to extract user identification information. When a JWT token is provided as input, the system automatically extracts the `sub` (subject) claim and uses it for user lookups.

### JWT Token Processing

**Token Format:** JWT tokens issued by Auth0

**Token Structure:**
```json
{
  "iss": "https://{{tenant}}.auth0.com/",
  "sub": "auth0|user123",
  "aud": "https://{{tenant}}.auth0.com/api/v2/",
  "iat": 1759751739,
  "exp": 1759755339,
  "scope": "read:current_user",
  "azp": "O8sQ4Jbr3At8buVR3IkrTRlejPZFWenI"
}
```

### Token Processing Flow

1. **Token Validation**: Validates the JWT token signature and expiration
2. **Sub Extraction**: Extracts the `sub` claim from the token payload
3. **User Lookup**: Uses the extracted `sub` value for direct user lookup via Auth0 Management API
4. **Auth0 API Call**: Performs direct user lookup using the `sub` identifier
5. **User Data Retrieval**: Returns user metadata from Auth0

### Auth0 Management API Integration

**Canonical Lookup (Recommended):**
```http
GET /api/v2/users/{sub}
```

**Search Lookup (Convenience):**
```http
GET /api/v2/users?q=identities.user_id:{username} AND identities.connection:Username-Password-Authentication
```

### Important Notes

- **JWT Signature Validation**: Full JWT signature validation is performed using Auth0's public keys
- **Token Expiration**: JWT tokens are validated for expiration and freshness
- **Auth0 Management API**: Uses Auth0's Management API for user data retrieval
