# Mock User Infrastructure

This package provides a mock implementation of the user infrastructure for testing and development purposes.

## Overview

The mock user system provides a simple in-memory storage solution for user data with two data sources:
1. **YAML File** (Primary): Embedded YAML file with three fantasy-themed users with complete profile data
2. **Hardcoded Users** (Fallback): Same user data defined in code as backup

## Features

- **In-memory storage**: Fast, stateful mock operations during runtime
- **YAML primary source**: Embedded YAML file with three predefined users with fantasy names for consistent testing
- **Hardcoded fallback**: Code-defined users as backup data source
- **Dual-key lookup**: Users can be found by either username or primary email
- **PATCH-style updates**: Only non-empty/non-nil fields are updated
- **Comprehensive logging**: Detailed logging for debugging and monitoring

## Mock Users

The system includes three hardcoded users with fantasy names:

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

## Data Sources

### Primary: Embedded YAML File
The primary data source is the embedded `users.yaml` file loaded using Go's `//go:embed` directive. This allows easy modification of user data without changing code, making it ideal for different testing scenarios.

### Fallback: Hardcoded Users
If the YAML file is unavailable or fails to parse, the system falls back to the `getHardcodedUsers()` function that returns the same three predefined users with complete profile information. This ensures the system always has data available.

```yaml
users:
  - token: "mock-token-zephyr-001"
    user_id: "user-001"
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
- Development environments where real Auth0 integration isn't needed
- Demo environments requiring realistic but fake user profiles

## Extending the System

To add more mock users:
1. Update the `users.yaml` file with the new user data (primary source)
2. Add them to the `getHardcodedUsers()` function (fallback source)
3. Update this README with the new user information

The system will automatically handle the additional users without code changes to the core logic. Since YAML is the primary source, you can modify user data by just editing the YAML file.
