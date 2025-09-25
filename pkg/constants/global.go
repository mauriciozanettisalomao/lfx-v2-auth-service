// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package constants

const (

	// ServiceName is the name of the auth service
	ServiceName = "lfx-v2-auth-service"

	// UserRepositoryTypeEnvKey is the environment variable key for the user repository type
	UserRepositoryTypeEnvKey = "USER_REPOSITORY_TYPE"

	// UserRepositoryTypeMock is the value for the mock user repository type
	UserRepositoryTypeMock = "mock"

	// UserRepositoryTypeAuth0 is the value for the Auth0 user repository type
	UserRepositoryTypeAuth0 = "auth0"

	// UserRepositoryTypeAuthelia is the value for the Authelia user repository type
	UserRepositoryTypeAuthelia = "authelia"

	// Auth0 Management API configuration
	// Auth0TenantEnvKey is the environment variable key for the Auth0 tenant
	Auth0TenantEnvKey = "AUTH0_TENANT"

	// Auth0DomainEnvKey is the environment variable key for the Auth0 domain
	Auth0DomainEnvKey = "AUTH0_DOMAIN"

	// Authelia configuration
	// AutheliaConfigMapNameEnvKey is the environment variable key for the ConfigMap name
	AutheliaConfigMapNameEnvKey = "AUTHELIA_CONFIGMAP_NAME"

	// AutheliaConfigMapNamespaceEnvKey is the environment variable key for the ConfigMap namespace
	AutheliaConfigMapNamespaceEnvKey = "AUTHELIA_CONFIGMAP_NAMESPACE"

	// AutheliaUsersFileKeyEnvKey is the environment variable key for the users file key in ConfigMap
	AutheliaUsersFileKeyEnvKey = "AUTHELIA_USERS_FILE_KEY"
)
