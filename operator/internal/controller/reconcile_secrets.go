/*
Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	opencodev1alpha1 "github.com/timothyclin/k8s-opencode/operator/api/v1alpha1"
)

const providerSecretsName = "provider-api-keys"

type APIKeys struct {
	Anthropic  string
	OpenAI     string
	OpenRouter string
}

func (r *OpenCodeWorkspaceReconciler) reconcileSecrets(ctx context.Context, workspace *opencodev1alpha1.OpenCodeWorkspace, namespaceName string) (*APIKeys, error) {
	apiKeys := make(map[string][]byte)
	result := &APIKeys{}

	if workspace.Spec.Providers.Anthropic.Enabled {
		ref := workspace.Spec.Providers.Anthropic.APIKeySecretRef
		if ref.Name != "" && ref.Namespace != "" {
			key, err := r.getSecretKey(ctx, ref.Namespace, ref.Name, "api-key")
			if err != nil {
				return nil, fmt.Errorf("failed to get Anthropic API key: %w", err)
			}
			apiKeys["ANTHROPIC_API_KEY"] = key
			result.Anthropic = string(key)
		}
	}

	if workspace.Spec.Providers.OpenAI.Enabled {
		ref := workspace.Spec.Providers.OpenAI.APIKeySecretRef
		if ref.Name != "" && ref.Namespace != "" {
			key, err := r.getSecretKey(ctx, ref.Namespace, ref.Name, "api-key")
			if err != nil {
				return nil, fmt.Errorf("failed to get OpenAI API key: %w", err)
			}
			apiKeys["OPENAI_API_KEY"] = key
			result.OpenAI = string(key)
		}
	}

	if workspace.Spec.Providers.OpenRouter.Enabled {
		ref := workspace.Spec.Providers.OpenRouter.APIKeySecretRef
		if ref.Name != "" && ref.Namespace != "" {
			key, err := r.getSecretKey(ctx, ref.Namespace, ref.Name, "api-key")
			if err != nil {
				return nil, fmt.Errorf("failed to get OpenRouter API key: %w", err)
			}
			apiKeys["OPENROUTER_API_KEY"] = key
			result.OpenRouter = string(key)
		}
	}

	if len(apiKeys) == 0 {
		return result, nil
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      providerSecretsName,
			Namespace: namespaceName,
		},
	}

	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, secret, func() error {
		if err := controllerutil.SetOwnerReference(workspace, secret, r.Scheme); err != nil {
			return err
		}

		secret.Type = corev1.SecretTypeOpaque
		secret.Data = apiKeys

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to reconcile secrets %q: %w", providerSecretsName, err)
	}

	return result, nil
}

// getSecretKey retrieves a specific key from a secret in another namespace.
func (r *OpenCodeWorkspaceReconciler) getSecretKey(ctx context.Context, namespace, name, key string) ([]byte, error) {
	secret := &corev1.Secret{}
	err := r.Get(ctx, types.NamespacedName{Namespace: namespace, Name: name}, secret)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, fmt.Errorf("secret %s/%s not found", namespace, name)
		}
		return nil, err
	}

	value, ok := secret.Data[key]
	if !ok {
		return nil, fmt.Errorf("key %q not found in secret %s/%s", key, namespace, name)
	}

	return value, nil
}
