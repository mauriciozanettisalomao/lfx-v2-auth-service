# LFX v2 Auth Service

A NATS-based authentication and user management microservice for the LFX v2 platform. This service provides an abstraction layer between applications and identity providers (Auth0 and Authelia).

## Overview

The LFX v2 Auth Service provides authentication and profile access in the v2 Platform, serving as an abstraction layer between applications and identity providers (Auth0 and Authelia). This service enables user management, profile updates, email/social account linking, and user discovery while maintaining compatibility across different deployment environments.

The service operates as a NATS-based microservice, responding to request/reply patterns on specific subjects.

### Prerequisites
- Go 1.24.5+
- NATS server
- Auth0 configuration (optional, defaults to mock mode)

### Installation

```bash
├── .github/                        # Github files
│   └── workflows/                  # Github Action workflow files
├── charts/                         # Helm charts for running the service in kubernetes
├── cmd/                            # Services (main packages)
│   └── server/                     # Auth service code
├── internal/                       # Internal service packages
│   ├── domain/                     # Domain logic layer (business logic)
│   │   ├── model/                  # Domain models and entities
│   │   └── port/                   # Repository and service interfaces
│   ├── service/                    # Service logic layer (use cases)
│   └── infrastructure/             # Infrastructure layer
├── pkg/                            # Shared packages
│   ├── constants/                  # Application constants
│   ├── converters/                 # Data conversion utilities
│   ├── errors/                     # Error handling utilities
│   ├── httpclient/                 # HTTP client utilities
│   └── log/                        # Logging utilities
└── README.md                       # This documentation
```

## Usage

### NATS Request/Reply Pattern

The LFX v2 Auth Service operates as a NATS-based microservice that responds to request/reply patterns on specific subjects. The service provides user management capabilities through NATS messaging.

#### Email to Username Lookup

To look up a username by email address, send a NATS request to the following subject:

**Subject:** `lfx.auth-service.email_to_username`  
**Pattern:** Request/Reply

##### Request Payload

The request payload should be a plain text email address (no JSON wrapping required):

```
user@example.com
```

##### Reply

The service returns the username as plain text if the email is found:

**Success Reply:**
```
john.doe
```

**Error Reply:**
```json
{
  "error": "user not found"
}
```

##### Example using NATS CLI

```bash
# Look up username by email
nats request lfx.auth-service.email_to_username zephyr.stormwind@mythicaltech.io

# Expected response: zephyr.stormwind
```

**Important Notes:**
- This service searches for users by their **primary email** only
- Linked/alternate email addresses are **not** supported for lookup
- The service works with both Auth0 and mock repositories based on configuration

#### User Update Operations

To update a user profile, send a NATS request to the following subject:

**Subject:** `lfx.auth-service.user_metadata.update`  
**Pattern:** Request/Reply

##### Request Payload

The request payload must be a JSON object containing the user data to update. The `token` field is **required** for authentication and authorization.

```json
{
  "token": "eyJhbG...",
  "user_id": "auth0|zephyr001",
  "username": "zephyr.stormwind",
  "primary_email": "zephyr.stormwind@mythicaltech.io",
  "user_metadata": {
    "name": "Zephyr Stormwind",
    "given_name": "Zephyr",
    "family_name": "Stormwind",
    "job_title": "Cloud Architect",
    "organization": "Mythical Tech Solutions",
    "country": "Aetheria",
    "state_province": "Skylands",
    "city": "Nimbus City",
    "address": "42 Celestial Tower, Cloud District",
    "postal_code": "90210",
    "phone_number": "+1-555-STORM-01",
    "t_shirt_size": "M",
    "picture": "https://avatars.mythicaltech.io/zephyr.jpg",
    "zoneinfo": "Aetheria/Skylands"
  }
}
```

##### Required Fields

- `token`: JWT authentication token (required for all requests)
- `user_metadata`: Object containing additional user profile information

##### Reply

The service returns a structured reply indicating success or failure:

**Success Reply:**
```json
{
  "success": true,
  "data": {
    "name": "Zephyr Stormwind",
    "given_name": "Zephyr",
    "family_name": "Stormwind",
    "job_title": "Cloud Architect",
    "organization": "Mythical Tech Solutions",
    "country": "Aetheria",
    "state_province": "Skylands",
    "city": "Nimbus City",
    "address": "42 Celestial Tower, Cloud District",
    "postal_code": "90210",
    "phone_number": "+1-555-STORM-01",
    "t_shirt_size": "M",
    "picture": "https://avatars.mythicaltech.io/zephyr.jpg",
    "zoneinfo": "Aetheria/Skylands"
  }
}
```

**Error Reply:**
```json
{
  "success": false,
  "error": "username is required"
}
```

##### Example using NATS CLI

```bash
# Update user profile
nats request lfx.auth-service.user_metadata.update '{
  "token": "eyJhbG...",
  "user_metadata": {
    "name": "Aurora Moonbeam",
    "job_title": "Senior DevOps Enchanter"
  }
}'
```

#### Configuration

##### NATS Configuration

The NATS client can be configured using environment variables:

- `NATS_URL`: NATS server URL (default: `nats://localhost:4222`)
- `NATS_TIMEOUT`: Request timeout duration (default: `10s`)
- `NATS_MAX_RECONNECT`: Maximum reconnection attempts (default: `3`)
- `NATS_RECONNECT_WAIT`: Time between reconnection attempts (default: `2s`)

##### Auth0 Configuration

The Auth0 integration can be configured using environment variables:

- `AUTH0_TENANT`: Auth0 tenant name (e.g., `"linuxfoundation"`, `"linuxfoundation-staging"`, `"linuxfoundation-dev"`)
  - **If not set, the service will automatically use mock/local behavior**
- `AUTH0_DOMAIN`: Auth0 domain for Management API calls (e.g., `"sso.linuxfoundation.org"`)
  - **If not set, defaults to `${AUTH0_TENANT}.auth0.com`**
- `USER_REPOSITORY_TYPE`: Set to `"auth0"` to use Auth0 integration, or `"mock"` for local development
  - **Defaults to `"auth0"` when `AUTH0_TENANT` is set, `"mock"` otherwise**

**Note:** When `AUTH0_DOMAIN` and `AUTH0_MANAGEMENT_TOKEN` are not set, the service will validate JWT tokens but won't make actual calls to Auth0's Management API.

## Releases

### Creating a Release

To create a new release of the auth service:

1. **Update the chart version** in `charts/lfx-v2-auth-service/Chart.yaml` prior to any project releases, or if any
   change is made to the chart manifests or configuration:
   ```yaml
   version: 0.2.0  # Increment this version
   appVersion: "latest"  # Keep this as "latest"
   ```

2. **After the pull request is merged**, create a GitHub release and choose the
   option for GitHub to also tag the repository. The tag must follow the format
   `v{version}` (e.g., `v0.2.0`). This tag does _not_ have to match the chart
   version: it is the version for the project release, which will dynamically
   update the `appVersion` in the released chart.

3. **The GitHub Actions workflow will automatically**:
   - Build and publish the container images (auth-service)
   - Package and publish the Helm chart to GitHub Pages
   - Publish the chart to GitHub Container Registry (GHCR)
   - Sign the chart with Cosign
   - Generate SLSA provenance

### Important Notes

- The `appVersion` in `Chart.yaml` should always remain `"latest"` in the committed code.
- During the release process, the `ko-build-tag.yaml` workflow automatically overrides the `appVersion` with the actual tag version (e.g., `v0.2.0` becomes `0.2.0`).
- Only update the chart `version` field when making releases - this represents the Helm chart version.
- The container image tags are automatically managed by the consolidated CI/CD pipeline using the git tag.
- Both container images (auth-service) and the Helm chart are published together in a single workflow.

## Development

To contribute to this repository:

1. Fork the repository
2. Commit your changes to a feature branch in your fork. Ensure your commits
   are signed with the [Developer Certificate of Origin
   (DCO)](https://developercertificate.org/).
   You can use the `git commit -s` command to sign your commits.
3. Ensure the chart version in `charts/lfx-v2-auth-service/Chart.yaml` has been
   updated following semantic version conventions if you are making changes to the chart.
4. Submit your pull request

## License

Copyright The Linux Foundation and each contributor to LFX.

This project’s source code is licensed under the MIT License. A copy of the
license is available in `LICENSE`.

This project’s documentation is licensed under the Creative Commons Attribution
4.0 International License \(CC-BY-4.0\). A copy of the license is available in
`LICENSE-docs`.