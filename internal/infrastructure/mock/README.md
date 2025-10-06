# Mock User Infrastructure

This package provides a mock implementation of the user infrastructure for testing and development purposes.

## Overview

The mock user system provides a simple in-memory storage solution for user data using an embedded YAML file, which can be modified as needed during the development phase. (Be careful when modifying and committing the file — currently, the data is fantasy-themed and not real or sensitive.)

## Features

- **In-memory storage**: Fast, stateful mock operations during runtime
- **YAML data source**: Embedded YAML file with five predefined users for consistent testing
- **JWT token support**: Parses JWT tokens and extracts the `sub` claim for user identification
- **PATCH-style updates**: Only non-empty/non-nil fields are updated
- **Comprehensive logging**: Detailed logging for debugging and monitoring

## Mock Users

The system includes five users defined in the YAML file:

### 1. Zephyr Stormwind
- **Username**: `zephyr.stormwind`
- **Email**: `zephyr.stormwind@mockdomain.com`
- **Role**: Cloud Architect at Mythical Tech Solutions
- **Location**: New York, USA
- **Timezone**: America/New_York

### 2. Aurora Moonbeam
- **Username**: `aurora.moonbeam`
- **Email**: `aurora.moonbeam@fantasycorp.io`
- **Role**: Senior DevOps Engineer at Enchanted Systems Ltd
- **Location**: London, UK
- **Timezone**: Europe/London

### 3. Phoenix Fireforge
- **Username**: `phoenix.fireforge`
- **Email**: `phoenix.fireforge@legendarydev.net`
- **Role**: Full Stack Wizard at Mythical Development Co
- **Location**: San Francisco, USA
- **Timezone**: America/Los_Angeles

## JWT Token Support

The mock implementation supports JWT (JSON Web Token) parsing to extract user identification information. When a JWT token is provided as input, the system automatically extracts the `sub` (subject) claim and uses it for user lookups.

### JWT Token Generation

For testing purposes, you can generate JWT tokens using the [JWT.io](https://www.jwt.io) online tool:

1. Go to [https://www.jwt.io](https://www.jwt.io)
2. Navigate to the **JWT Encoder** tab
3. Use the following configuration:

**Header:**
```json
{
  "alg": "HS256",
  "typ": "JWT"
}
```

**Payload:**
```json
{
  "sub": "auth0|zephyr001",
  "name": "Zephyr Stormwind",
  "admin": true,
  "iat": 1516239022
}
```

**Secret:**
```
a-string-secret-at-least-256-bits-long
```

### JWT Processing Flow

1. **Token Detection**: The system detects JWT tokens by checking if the input starts with `"eyJ"` (standard JWT header)
2. **Parsing**: Uses `github.com/golang-jwt/jwt/v5` to parse the token without signature verification
3. **Sub Extraction**: Extracts the `sub` claim from the token payload
4. **Lookup Strategy**: Uses the extracted `sub` value to determine the appropriate lookup strategy:
   - If `sub` contains `|` → Canonical lookup (direct user ID match)
   - If `sub` contains `@` → Email search
   - Otherwise → Username search
5. **Fallback**: If JWT parsing fails, treats the input as a regular string

### Important Notes

- **No Signature Validation**: For simplicity in the mock environment, JWT signature validation is **not** performed. This avoids overcomplicating the development flow while still providing realistic JWT parsing behavior.
- **Development Only**: This JWT parsing is intended for development and testing purposes only. Production implementations should include proper signature validation.
- **Flexible Input**: The system gracefully handles both JWT tokens and regular string inputs (usernames, emails, user IDs).

## Data Source

### Embedded YAML File
The data source is the embedded `users.yaml` file loaded using Go's `//go:embed` directive. This allows easy modification of user data without changing code, making it ideal for different testing scenarios.

```yaml
users:
  - token: "mock-token-zephyr-001"
    user_id: "user-001"
    sub: "provider|user-001"
    username: "zephyr.stormwind"
    primary_email: "zephyr.stormwind@mockdomain.com"
    user_metadata:
      picture: "https://api.dicebear.com/7.x/avataaars/svg?seed=zephyr"
      zoneinfo: "America/New_York"
    # ... complete user data
```

## Implementation Details

### Key Features

1. **Dual-key Storage**: Each user is stored twice in the internal map - once with username as key and once with email as key, allowing flexible lookups.

2. **PATCH Semantics**: Updates only modify fields that are provided (non-empty strings, non-nil pointers), preserving existing data for unspecified fields.

3. **Embedded Files**: The YAML file is embedded into the binary at compile time using `//go:embed`, ensuring the fallback data is always available.

## File Structure

```
internal/infrastructure/mock/
├── README.md           # This documentation
├── user.go            # Main mock implementation
└── users.yaml         # Embedded fallback user data
```

## Development Notes

- Users have fantasy names to avoid confusion with real user data
- All user data is fake and safe for development/testing
- The system supports the full User and UserMetadata model structure

## Testing

The mock system is perfect for:
- Unit tests requiring predictable user data
- Integration tests needing consistent user scenarios
- JWT token parsing and validation testing
- Development environments where real Auth0 integration isn't needed
- Demo environments requiring realistic but fake user profiles

## Extending the System

To add more mock users:
1. Update the `users.yaml` file with the new user data
2. Update this README with the new user information

The system will automatically handle the additional users without code changes to the core logic. You can modify user data by just editing the YAML file.
