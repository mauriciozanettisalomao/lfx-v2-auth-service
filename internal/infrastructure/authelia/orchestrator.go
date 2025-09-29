// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package authelia

import (
	"context"

	"github.com/linuxfoundation/lfx-v2-auth-service/internal/infrastructure/k8s"
	"github.com/linuxfoundation/lfx-v2-auth-service/pkg/errors"
)

// UserOrchestrator defines the behavior of the orchestrator which is responsible
// for syncing users within the environment
//
// They are required only in the Authelia context
type internalOrchestrator interface {
	LoadUsersOrigin(ctx context.Context) (map[string]*any, error)
	UpdateOrigin(ctx context.Context, yamlData []byte) error
	RestartOrigin(ctx context.Context) error
	UpdateSecrets(ctx context.Context, secretData map[string][]byte) error
}

type k8sOrchestrator struct {
	config       map[string]string
	orchestrator *k8s.K8sOrchestrator
}

func (k *k8sOrchestrator) LoadUsersOrigin(ctx context.Context) (map[string]*any, error) {

	usersFileKey := k.config["users_file_key"]
	if usersFileKey == "" {
		return nil, errors.NewUnexpected("users file key is required")
	}

	usersFile, errGetUsersFile := k.orchestrator.Get(ctx, func() (any, error) {
		return k.orchestrator.ConfigMap(ctx, usersFileKey)
	})
	if errGetUsersFile != nil {
		return nil, errGetUsersFile
	}

	if data, ok := usersFile.(map[string]*any); !ok || data == nil {
		return data, nil
	}

	return nil, errors.NewUnexpected("invalid data type")

}

func (k *k8sOrchestrator) UpdateOrigin(ctx context.Context, yamlData []byte) error {
	return nil
}

func (k *k8sOrchestrator) RestartOrigin(ctx context.Context) error {
	return nil
}

func (k *k8sOrchestrator) UpdateSecrets(ctx context.Context, secretData map[string][]byte) error {
	return nil
}

func newK8sUserOrchestrator(ctx context.Context, config map[string]string) (internalOrchestrator, error) {

	k := &k8sOrchestrator{}

	orchestrator, errNewK8sOrchestrator := k8s.NewK8sOrchestrator(ctx, config)
	if errNewK8sOrchestrator != nil {
		return nil, errNewK8sOrchestrator
	}
	k.orchestrator = orchestrator

	return k, nil
}
