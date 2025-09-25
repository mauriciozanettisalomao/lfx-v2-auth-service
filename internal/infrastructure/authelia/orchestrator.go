// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package authelia

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/linuxfoundation/lfx-v2-auth-service/internal/domain/model"
	"github.com/linuxfoundation/lfx-v2-auth-service/pkg/converters"
	"github.com/linuxfoundation/lfx-v2-auth-service/pkg/errors"
	"gopkg.in/yaml.v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

// UserOrchestrator defines the interface for user writer orchestrator operations
// it's used to sync the authelia data in the orchestrator environment (k8s)
type UserOrchestrator interface {
	LoadUsersFromConfigMap(ctx context.Context) (map[string]*AutheliaUser, error)
	RestartAutheliaDaemonSet(ctx context.Context) error
	UpdateSecrets(ctx context.Context, secretData map[string][]byte) error
	UpdateConfigMap(ctx context.Context, yamlData []byte) error
}

type k8sOrchestrator struct {
	k8sClient kubernetes.Interface
	config    Config
}

// LoadUsersFromConfigMap loads existing users from the ConfigMap
func (k *k8sOrchestrator) LoadUsersFromConfigMap(ctx context.Context) (map[string]*AutheliaUser, error) {
	slog.DebugContext(ctx, "loading users from ConfigMap",
		"configmap", k.config.ConfigMapName,
		"namespace", k.config.ConfigMapNamespace)

	// Get ConfigMap
	configMap, err := k.k8sClient.CoreV1().ConfigMaps(k.config.ConfigMapNamespace).Get(
		ctx, k.config.ConfigMapName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get ConfigMap: %w", err)
	}

	usersYAML, exists := configMap.Data[k.config.UsersFileKey]
	if !exists {
		slog.InfoContext(ctx, "users file not found in ConfigMap, starting fresh")
		return make(map[string]*AutheliaUser), nil
	}

	// Parse Authelia users YAML
	var autheliaDB AutheliaUsersDatabase
	if err := yaml.Unmarshal([]byte(usersYAML), &autheliaDB); err != nil {
		return nil, fmt.Errorf("failed to parse Authelia YAML: %w", err)
	}

	// Convert to AutheliaUser format
	users := make(map[string]*AutheliaUser)

	for username, configUser := range autheliaDB.Users {

		// Build user metadata from structured fields
		userMetadata := &model.UserMetadata{}
		if configUser.DisplayName != "" {
			userMetadata.Name = &configUser.DisplayName
		}

		// Convert metadata map to structured fields
		if configUser.UserMetadata != nil {
			userMetadata.Picture = converters.StringPtr(configUser.UserMetadata["picture"])
			userMetadata.Zoneinfo = converters.StringPtr(configUser.UserMetadata["zoneinfo"])
			userMetadata.Name = converters.StringPtr(configUser.UserMetadata["name"])
			userMetadata.GivenName = converters.StringPtr(configUser.UserMetadata["given_name"])
			userMetadata.FamilyName = converters.StringPtr(configUser.UserMetadata["family_name"])
			userMetadata.JobTitle = converters.StringPtr(configUser.UserMetadata["job_title"])
			userMetadata.Organization = converters.StringPtr(configUser.UserMetadata["organization"])
			userMetadata.Country = converters.StringPtr(configUser.UserMetadata["country"])
			userMetadata.StateProvince = converters.StringPtr(configUser.UserMetadata["state_province"])
			userMetadata.City = converters.StringPtr(configUser.UserMetadata["city"])
			userMetadata.Address = converters.StringPtr(configUser.UserMetadata["address"])
			userMetadata.PostalCode = converters.StringPtr(configUser.UserMetadata["postal_code"])
			userMetadata.PhoneNumber = converters.StringPtr(configUser.UserMetadata["phone_number"])
			userMetadata.TShirtSize = converters.StringPtr(configUser.UserMetadata["t_shirt_size"])
		}

		// Create model.User
		modelUser := &model.User{
			UserID:       username,
			Username:     username,
			PrimaryEmail: configUser.Email,
			UserMetadata: userMetadata,
		}

		// Create AutheliaUser wrapper and set password information
		autheliaUser := newAutheliaUser(modelUser)
		autheliaUser.Password = configUser.Password // Keep existing bcrypt hash
		autheliaUser.CreatedAt = time.Now()
		autheliaUser.UpdatedAt = time.Now()

		// Store the AutheliaUser
		users[username] = autheliaUser
		slog.DebugContext(ctx, "loaded user from ConfigMap", "username", username)
	}

	return users, nil
}

// RestartAutheliaDaemonSet restarts the Authelia daemonset to pick up ConfigMap changes
func (k *k8sOrchestrator) RestartAutheliaDaemonSet(ctx context.Context) error {
	if k.k8sClient == nil {
		return fmt.Errorf("kubernetes client not available")
	}

	slog.InfoContext(ctx, "restarting Authelia daemonset",
		"daemonset", k.config.DaemonSetName,
		"namespace", k.config.DaemonSetNamespace)

	// Get the daemonset
	daemonSetClient := k.k8sClient.AppsV1().DaemonSets(k.config.DaemonSetNamespace)
	daemonSet, err := daemonSetClient.Get(ctx, k.config.DaemonSetName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get daemonset %s/%s: %w",
			k.config.DaemonSetNamespace, k.config.DaemonSetName, err)
	}

	// Add/update restart annotation to trigger rolling restart
	if daemonSet.Spec.Template.Annotations == nil {
		daemonSet.Spec.Template.Annotations = make(map[string]string)
	}

	// Use current timestamp as annotation value to trigger restart
	restartTime := time.Now().Format(time.RFC3339)
	daemonSet.Spec.Template.Annotations["kubectl.kubernetes.io/restartedAt"] = restartTime

	// Update the daemonset
	_, err = daemonSetClient.Update(ctx, daemonSet, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update daemonset %s/%s: %w",
			k.config.DaemonSetNamespace, k.config.DaemonSetName, err)
	}

	slog.InfoContext(ctx, "successfully triggered daemonset restart",
		"daemonset", k.config.DaemonSetName,
		"namespace", k.config.DaemonSetNamespace,
		"restart_time", restartTime)

	return nil
}

// UpdateSecrets updates only the specified entries in the secret
func (k *k8sOrchestrator) UpdateSecrets(ctx context.Context, changedEntries map[string][]byte) error {
	if k.k8sClient == nil {
		return fmt.Errorf("kubernetes client not available")
	}

	// Skip if no changes
	if len(changedEntries) == 0 {
		slog.DebugContext(ctx, "no changes to Secret, skipping update")
		return nil
	}

	// Get the existing Secret
	secretsClient := k.k8sClient.CoreV1().Secrets(k.config.SecretNamespace)
	existingSecret, err := secretsClient.Get(ctx, k.config.SecretName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get existing Secret: %w", err)
	}

	// Initialize data map if it doesn't exist
	if existingSecret.Data == nil {
		existingSecret.Data = make(map[string][]byte)
	}

	// Update changed entries
	for username, password := range changedEntries {
		existingSecret.Data[username] = password
	}

	// Update the secret
	_, err = secretsClient.Update(ctx, existingSecret, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update Secret: %w", err)
	}

	slog.InfoContext(ctx, "selectively updated Secret with user passwords",
		"secret", k.config.SecretName,
		"namespace", k.config.SecretNamespace,
		"updated_users", len(changedEntries),
		"total_users", len(existingSecret.Data))

	return nil
}

// UpdateConfigMap updates the ConfigMap with the given YAML data
func (k *k8sOrchestrator) UpdateConfigMap(ctx context.Context, yamlData []byte) error {

	if k.k8sClient == nil {
		return fmt.Errorf("kubernetes client not available")
	}

	slog.DebugContext(ctx, "updating ConfigMap with user data")

	// Get the existing ConfigMap
	configMap, err := k.k8sClient.CoreV1().ConfigMaps(k.config.ConfigMapNamespace).Get(
		ctx, k.config.ConfigMapName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get ConfigMap: %w", err)
	}

	if configMap.Data == nil {
		configMap.Data = make(map[string]string)
	}
	configMap.Data[k.config.UsersFileKey] = string(yamlData)

	_, err = k.k8sClient.CoreV1().ConfigMaps(k.config.ConfigMapNamespace).Update(
		ctx, configMap, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update ConfigMap: %w", err)
	}

	slog.InfoContext(ctx, "updated ConfigMap with user data",
		"configmap", k.config.ConfigMapName,
		"namespace", k.config.ConfigMapNamespace)

	return nil
}

func (k *k8sOrchestrator) client(ctx context.Context) error {

	findConfig := func() (*rest.Config, error) {
		if _, exists := os.LookupEnv("KUBERNETES_SERVICE_HOST"); exists {
			slog.DebugContext(ctx, "using in-cluster Kubernetes config")
			return rest.InClusterConfig()
		}
		kubeconfigPath := os.Getenv("KUBECONFIG")
		if kubeconfigPath == "" {
			slog.DebugContext(ctx, "using local kubeconfig")
			if home := homedir.HomeDir(); home != "" {
				kubeconfigPath = filepath.Join(home, ".kube", "config")
			}
		}
		return clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	}

	k8sConfig, errFindConfig := findConfig()
	if errFindConfig != nil {
		return errors.NewUnexpected("failed to find Kubernetes config", errFindConfig)
	}

	// Create Kubernetes client if config was loaded successfully
	if k8sConfig != nil {
		k8sClient, errNewForConfig := kubernetes.NewForConfig(k8sConfig)
		if errNewForConfig != nil {
			return errors.NewUnexpected("failed to create Kubernetes client", errNewForConfig)
		}
		k.k8sClient = k8sClient
		return nil
	}

	return errors.NewUnexpected("failed to find Kubernetes config", errFindConfig)
}

// newK8sOrchestrator creates a new k8s orchestrator with auto-configured Kubernetes client
func newK8sOrchestrator(ctx context.Context, config Config) (UserOrchestrator, error) {

	k8s := &k8sOrchestrator{
		config: config,
	}

	err := k8s.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kubernetes client: %w", err)
	}

	return k8s, nil
}
