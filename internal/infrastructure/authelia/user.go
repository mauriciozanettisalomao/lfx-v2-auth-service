// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package authelia

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/linuxfoundation/lfx-v2-auth-service/internal/domain/model"
	"github.com/linuxfoundation/lfx-v2-auth-service/internal/domain/port"
	"github.com/linuxfoundation/lfx-v2-auth-service/internal/infrastructure/nats"
	"github.com/linuxfoundation/lfx-v2-auth-service/pkg/constants"
	errs "github.com/linuxfoundation/lfx-v2-auth-service/pkg/errors"
	"github.com/linuxfoundation/lfx-v2-auth-service/pkg/misc"

	"gopkg.in/yaml.v3"
)

// userWriter implements UserReaderWriter with pluggable storage and ConfigMap sync
type userWriter struct {
	config       Config
	storage      storageReaderWriter
	orchestrator UserOrchestrator
}

// userStorage creates the appropriate storage implementation based on configuration
func (u *userWriter) userStorage(ctx context.Context, natsClient *nats.NATSClient) error {
	// If NATS client is available and has the required KV store, use NATS storage
	if natsClient != nil {
		if _, exists := natsClient.GetKVStore(constants.KVBucketNameAutheliaUsers); exists {
			slog.DebugContext(ctx, "using NATS-based user storage for authelia users")
			storage, errNATSUserStorage := newNATSUserStorage(ctx, natsClient)
			if errNATSUserStorage != nil {
				return errs.NewUnexpected("failed to create NATS user storage", errNATSUserStorage)
			}
			u.storage = storage
			return nil
		}
	}
	return errs.NewUnexpected("no storage implementation available")
}

func (u *userWriter) userOrchestrator(ctx context.Context, config Config) error {
	orchestrator, err := newK8sOrchestrator(ctx, config)
	if err != nil {
		return errs.NewUnexpected("failed to create k8s orchestrator", err)
	}
	u.orchestrator = orchestrator
	return nil
}

// UserSyncResult represents the result of comparing users from different sources
// It contains information about what changes were detected and what action should be taken
type UserSyncResult struct {
	Username        string        // The username being compared
	ConfigMapUser   *AutheliaUser // User data from ConfigMap (nil if not present)
	NATSUser        *AutheliaUser // User data from NATS KV (nil if not present)
	EmailChanged    bool          // True if email addresses differ between sources
	PasswordChanged bool          // True if password appears to have changed
	Action          string        // Action to take: "sync_to_configmap", "sync_to_nats", "no_change", "password_updated"
}

// compareUsers compares a user from ConfigMap with a user from NATS KV
// and determines what synchronization action should be taken.
//
// Logic:
// - If user exists only in one source: sync to the other source
// - If user exists in both sources: compare email and password changes only
func (u *userWriter) compareUsers(configMapUser *AutheliaUser, natsUser *AutheliaUser) UserSyncResult {
	username := ""
	if configMapUser != nil {
		username = configMapUser.User.Username
	} else if natsUser != nil {
		username = natsUser.User.Username
	}

	result := UserSyncResult{
		Username:      username,
		ConfigMapUser: configMapUser,
		NATSUser:      natsUser,
		Action:        "no_change",
	}

	// If user exists only in one source
	if configMapUser == nil && natsUser != nil {
		result.Action = "sync_to_configmap"
		return result
	}
	if natsUser == nil && configMapUser != nil {
		result.Action = "sync_to_nats"
		return result
	}
	if configMapUser == nil && natsUser == nil {
		result.Action = "no_change"
		return result
	}

	// Both users exist - compare email and password only
	emailChanged := configMapUser.User.PrimaryEmail != natsUser.User.PrimaryEmail
	passwordChanged := configMapUser.Password != natsUser.Password

	result.EmailChanged = emailChanged
	result.PasswordChanged = passwordChanged

	// Determine sync action based on changes
	if passwordChanged {
		result.Action = "password_updated"
	}

	return result
}

// mergeSources synchronizes users between ConfigMap and NATS KV
// This is the main synchronization function that:
// 1. Compares all users from both sources
// 2. Determines what sync actions are needed
// 3. Creates a merged view of all users
// 4. Performs necessary updates to NATS KV
// 5. Returns the merged user list for ConfigMap sync
func (u *userWriter) mergeSources(ctx context.Context, configMapUsers map[string]*AutheliaUser, natsUsers map[string]*AutheliaUser) error {

	slog.DebugContext(ctx, "synchronizing users between Orchestrator and Storage",
		"configmap_users", len(configMapUsers),
		"nats_users", len(natsUsers))

	// Create a set of all usernames from both sources
	allUsernames := make(map[string]bool)
	for username := range configMapUsers {
		allUsernames[username] = true
	}
	for username := range natsUsers {
		allUsernames[username] = true
	}

	mergedUsers := make(map[string]*AutheliaUser)
	syncResults := make([]UserSyncResult, 0)

	// Compare each user
	for username := range allUsernames {
		configMapUser := configMapUsers[username]
		natsUser := natsUsers[username]

		result := u.compareUsers(configMapUser, natsUser)
		syncResults = append(syncResults, result)

		// Apply sync action
		switch result.Action {
		case "sync_to_configmap":
			// Use NATS user as source of truth
			if natsUser != nil {
				mergedUsers[username] = natsUser
				slog.InfoContext(ctx, "syncing user from NATS to ConfigMap",
					"username", username,
					"email_changed", result.EmailChanged,
					"password_changed", result.PasswordChanged)

				// Note: The actual ConfigMap sync will happen in the main initialization
				// when syncToConfigMap is called after all users are processed
			}

		case "sync_to_nats":
			// Use ConfigMap user as source of truth, but preserve PlainPassword from NATS if available
			if configMapUser != nil {

				mergedUsers[username] = configMapUser
				slog.InfoContext(ctx, "syncing user from ConfigMap to NATS",
					"username", username,
					"email_changed", result.EmailChanged,
					"password_changed", result.PasswordChanged)

				// Update NATS with ConfigMap data
				if err := u.storage.SetUser(ctx, username, configMapUser); err != nil {
					slog.ErrorContext(ctx, "failed to sync user to NATS",
						"username", username, "error", err)
				}
			}

		case "no_change":
			// No sync needed, use NATS if available (it has complete password info), otherwise ConfigMap
			if natsUser != nil {
				mergedUsers[username] = natsUser
			} else if configMapUser != nil {
				mergedUsers[username] = configMapUser
			}

		case "password_updated":
			slog.DebugContext(ctx, "user needs update, preferring NATS version",
				"username", username)
			// Prefer ConfigMap since it has complete password information
			if configMapUser != nil {
				mergedUsers[username] = natsUser
			} else if natsUser != nil {
				mergedUsers[username] = configMapUser
			}

		}
		mergedUsers[username].SetActionNeeded(result.Action)
	}

	var errorList []error
	if err := u.syncOrchestrator(ctx, mergedUsers); err != nil {
		slog.Error("failed to sync users to orchestrator", "error", err)
		errorList = append(errorList, err)
	}
	if err := u.orchestrator.RestartAutheliaDaemonSet(ctx); err != nil {
		slog.Error("failed initial DaemonSet sync, continuing anyway", "error", err)
		errorList = append(errorList, err)
	}

	// Count different types of sync actions for reporting
	syncStats := map[string]int{
		"sync_to_configmap": 0,
		"sync_to_nats":      0,
		"no_change":         0,
		"password_updated":  0,
	}

	for _, result := range syncResults {
		syncStats[result.Action]++
	}

	slog.InfoContext(ctx, "user synchronization completed",
		"errors_length", len(errorList),
		"merged_users", len(mergedUsers),
		"sync_to_configmap", syncStats["sync_to_configmap"],
		"sync_to_nats", syncStats["sync_to_nats"],
		"no_change", syncStats["no_change"],
		"password_updated", syncStats["password_updated"],
	)

	return errors.Join(errorList...)
}

func (u *userWriter) syncUsers(ctx context.Context) error {

	// Load existing users from ConfigMap if orchestrator is available
	configMapUsers := make(map[string]*AutheliaUser)
	if u.orchestrator != nil {
		existingUsers, err := u.orchestrator.LoadUsersFromConfigMap(ctx)
		if err != nil {
			slog.Warn("failed to load existing users from ConfigMap, so no users will be loaded from ConfigMap",
				"error", err,
			)

		}
		configMapUsers = existingUsers
	}

	// Load existing users from NATS KV if storage is available
	natsUsers := make(map[string]*AutheliaUser)
	if u.storage != nil {
		existingNATSUsers, err := u.storage.ListUsers(ctx)
		if err != nil {
			slog.Warn("failed to load existing users from NATS KV",
				"error", err)
		}
		natsUsers = existingNATSUsers
	}

	// Synchronize users between ConfigMap and NATS KV
	return u.mergeSources(ctx, configMapUsers, natsUsers)
}

// NewAutheliaUserReaderWriter creates a new Authelia repository with optional NATS client
func NewAutheliaUserReaderWriter(ctx context.Context, config Config, natsClient *nats.NATSClient) (port.UserReaderWriter, error) {
	// Set defaults
	if config.ConfigMapName == "" {
		config.ConfigMapName = "authelia-users"
		slog.InfoContext(ctx, "using default ConfigMap name", "name", config.ConfigMapName)
	}
	if config.ConfigMapNamespace == "" {
		config.ConfigMapNamespace = "lfx"
	}
	if config.UsersFileKey == "" {
		config.UsersFileKey = "users_database.yml"
	}
	if config.SecretName == "" {
		config.SecretName = "authelia-users" // Same name as ConfigMap
	}
	if config.SecretNamespace == "" {
		config.SecretNamespace = config.ConfigMapNamespace // Same namespace as ConfigMap
	}
	if config.DaemonSetName == "" {
		config.DaemonSetName = "lfx-platform-authelia"
	}
	if config.DaemonSetNamespace == "" {
		config.DaemonSetNamespace = config.ConfigMapNamespace // Same namespace as ConfigMap
	}

	u := &userWriter{
		config: config,
	}

	// Initialize storage
	errUserStorage := u.userStorage(ctx, natsClient)
	if errUserStorage != nil {
		slog.ErrorContext(ctx, "failed to initialize storage", "error", errUserStorage)
		return nil, errUserStorage
	}

	// Initialize orchestrator
	errUserOrchestrator := u.userOrchestrator(ctx, config)
	if errUserOrchestrator != nil {
		slog.ErrorContext(ctx, "failed to create orchestrator", "error", errUserOrchestrator)
		return nil, errUserOrchestrator
	}

	errSyncUsers := u.syncUsers(ctx)
	if errSyncUsers != nil {
		slog.Warn("failed to sync from storage to orchestrator", "error", errSyncUsers)
	}

	return u, nil
}

// GetUser retrieves a user from storage
func (a *userWriter) GetUser(ctx context.Context, user *model.User) (*model.User, error) {
	username := a.getUsernameFromModel(user)
	if username == "" {
		return nil, errs.NewValidation("username is required")
	}

	slog.DebugContext(ctx, "getting user from storage", "username", username)

	autheliaUser, err := a.storage.GetUser(ctx, username)
	if err != nil {
		return nil, err
	}

	return autheliaUser.ToModelUser(), nil
}

// UpdateUser updates a user in storage and syncs to ConfigMap
func (a *userWriter) UpdateUser(ctx context.Context, user *model.User) (*model.User, error) {
	if user.UserMetadata == nil {
		return nil, errs.NewValidation("user_metadata is required for update")
	}

	return nil, nil
}

// syncOrchestrator syncs referenced users to the orchestrator
func (a *userWriter) syncOrchestrator(ctx context.Context, users map[string]*AutheliaUser) error {
	if a.orchestrator == nil {
		return fmt.Errorf("orchestrator not available")
	}

	slog.DebugContext(ctx, "syncing users from storage to ConfigMap")

	// Build Authelia users database from storage
	autheliaDB := AutheliaUsersDatabase{
		Users: make(map[string]AutheliaConfigUser),
	}

	// for the password updates, we need to update the secrets
	// so even if the storage has a password, we need to generate a new one
	changedSecretsEntries := make(map[string][]byte)
	for username, autheliaUser := range users {

		// If the user needs to be synced to the ConfigMap,
		// We need to regenerate the password, to get the plain text password
		// and update the secrets
		password := autheliaUser.Password
		if autheliaUser.ActionNeeded() == "sync_to_configmap" {
			plainPassword, bcryptHash, errGeneratePasswordPair := misc.GeneratePasswordPair(20)
			if errGeneratePasswordPair != nil {
				slog.WarnContext(ctx, "failed to generate password for user",
					"username", username, "error", errGeneratePasswordPair)
				continue
			}
			autheliaUser.Password = bcryptHash
			changedSecretsEntries[username] = []byte(plainPassword)

			// Update the user in storage with the new password information
			if err := a.storage.SetUser(ctx, username, autheliaUser); err != nil {
				slog.WarnContext(ctx, "failed to update user with generated password",
					"username", username, "error", err)
			}
			password = autheliaUser.Password

		}

		displayName := username
		if autheliaUser.DisplayName != "" {
			displayName = autheliaUser.DisplayName
		}
		if autheliaUser.User.UserMetadata != nil && autheliaUser.User.UserMetadata.Name != nil {
			displayName = *autheliaUser.User.UserMetadata.Name
		}

		email := autheliaUser.User.PrimaryEmail

		// Debug logging for email field
		slog.DebugContext(ctx, "processing user for ConfigMap sync",
			"username", username,
			"email", email,
			"display_name", displayName)

		// Safeguard: If email is empty, this indicates a bug in our sync logic
		if email == "" {
			slog.WarnContext(ctx, "user has empty email during ConfigMap sync - this may indicate a sync bug",
				"username", username)
		}

		// Create a simple config user that maintains the original Authelia structure
		configUser := AutheliaConfigUser{
			DisplayName: displayName,
			Password:    password,
			Email:       email,
		}

		// Only add user_metadata if there are additional metadata fields beyond name and email
		if autheliaUser.User.UserMetadata != nil {
			userMetadata := make(map[string]string)

			// Add all metadata fields except name (which is already in displayname)
			if autheliaUser.User.UserMetadata.Picture != nil {
				userMetadata["picture"] = *autheliaUser.User.UserMetadata.Picture
			}
			if autheliaUser.User.UserMetadata.Zoneinfo != nil {
				userMetadata["zoneinfo"] = *autheliaUser.User.UserMetadata.Zoneinfo
			}
			if autheliaUser.User.UserMetadata.GivenName != nil {
				userMetadata["given_name"] = *autheliaUser.User.UserMetadata.GivenName
			}
			if autheliaUser.User.UserMetadata.FamilyName != nil {
				userMetadata["family_name"] = *autheliaUser.User.UserMetadata.FamilyName
			}
			if autheliaUser.User.UserMetadata.JobTitle != nil {
				userMetadata["job_title"] = *autheliaUser.User.UserMetadata.JobTitle
			}
			if autheliaUser.User.UserMetadata.Organization != nil {
				userMetadata["organization"] = *autheliaUser.User.UserMetadata.Organization
			}
			if autheliaUser.User.UserMetadata.Country != nil {
				userMetadata["country"] = *autheliaUser.User.UserMetadata.Country
			}
			if autheliaUser.User.UserMetadata.StateProvince != nil {
				userMetadata["state_province"] = *autheliaUser.User.UserMetadata.StateProvince
			}
			if autheliaUser.User.UserMetadata.City != nil {
				userMetadata["city"] = *autheliaUser.User.UserMetadata.City
			}
			if autheliaUser.User.UserMetadata.Address != nil {
				userMetadata["address"] = *autheliaUser.User.UserMetadata.Address
			}
			if autheliaUser.User.UserMetadata.PostalCode != nil {
				userMetadata["postal_code"] = *autheliaUser.User.UserMetadata.PostalCode
			}
			if autheliaUser.User.UserMetadata.PhoneNumber != nil {
				userMetadata["phone_number"] = *autheliaUser.User.UserMetadata.PhoneNumber
			}
			if autheliaUser.User.UserMetadata.TShirtSize != nil {
				userMetadata["t_shirt_size"] = *autheliaUser.User.UserMetadata.TShirtSize
			}

			// Only set UserMetadata if there are additional fields
			if len(userMetadata) > 0 {
				configUser.UserMetadata = userMetadata
			}
		}

		autheliaDB.Users[username] = configUser
	}
	errUpdateSecrets := a.orchestrator.UpdateSecrets(ctx, changedSecretsEntries)
	if errUpdateSecrets != nil {
		return errs.NewUnexpected("failed to update Secrets", errUpdateSecrets)
	}

	// Marshal to YAML with 2-space indentation
	var buf strings.Builder
	encoder := yaml.NewEncoder(&buf)
	encoder.SetIndent(2)
	if err := encoder.Encode(autheliaDB); err != nil {
		return fmt.Errorf("failed to marshal YAML: %w", err)
	}
	encoder.Close()
	yamlData := []byte(buf.String())

	// Use orchestrator to update the ConfigMap
	if err := a.orchestrator.UpdateConfigMap(ctx, yamlData); err != nil {
		return fmt.Errorf("failed to update ConfigMap: %w", err)
	}

	slog.InfoContext(ctx, "synced users to ConfigMap", "users", len(autheliaDB.Users))
	return nil
}

// getUsernameFromModel returns the username from the model.User
// in the future, we might want to decode the token to get the username
func (a *userWriter) getUsernameFromModel(user *model.User) string {
	if user.Username != "" {
		return user.Username
	}
	if user.UserID != "" {
		return user.UserID
	}
	return ""
}

// mergeUserUpdates merges updates into an existing user
func (a *userWriter) mergeUserUpdates(existing *model.User, updates *model.User) *model.User {
	// Start with a copy of the existing user
	result := &model.User{
		UserID:       existing.UserID,
		Username:     existing.Username,
		PrimaryEmail: existing.PrimaryEmail,
		UserMetadata: &model.UserMetadata{},
	}

	// Copy existing metadata
	if existing.UserMetadata != nil {
		if existing.UserMetadata.Name != nil {
			name := *existing.UserMetadata.Name
			result.UserMetadata.Name = &name
		}
		if existing.UserMetadata.JobTitle != nil {
			jobTitle := *existing.UserMetadata.JobTitle
			result.UserMetadata.JobTitle = &jobTitle
		}
		if existing.UserMetadata.Organization != nil {
			org := *existing.UserMetadata.Organization
			result.UserMetadata.Organization = &org
		}
		if existing.UserMetadata.Country != nil {
			country := *existing.UserMetadata.Country
			result.UserMetadata.Country = &country
		}
		if existing.UserMetadata.City != nil {
			city := *existing.UserMetadata.City
			result.UserMetadata.City = &city
		}
	}

	// Apply updates
	if updates.PrimaryEmail != "" {
		result.PrimaryEmail = updates.PrimaryEmail
	}

	if updates.UserMetadata != nil {
		if updates.UserMetadata.Name != nil {
			result.UserMetadata.Name = updates.UserMetadata.Name
		}
		if updates.UserMetadata.JobTitle != nil {
			result.UserMetadata.JobTitle = updates.UserMetadata.JobTitle
		}
		if updates.UserMetadata.Organization != nil {
			result.UserMetadata.Organization = updates.UserMetadata.Organization
		}
		if updates.UserMetadata.Country != nil {
			result.UserMetadata.Country = updates.UserMetadata.Country
		}
		if updates.UserMetadata.City != nil {
			result.UserMetadata.City = updates.UserMetadata.City
		}
	}

	return result
}
