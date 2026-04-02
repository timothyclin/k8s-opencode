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
	"encoding/json"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	opencodev1alpha1 "github.com/timothyclin/k8s-opencode/operator/api/v1alpha1"
)

const opencodeConfigMapName = "opencode-config"

type opencodeConfig struct {
	Schema    string            `json:"$schema,omitempty"`
	Providers opencodeProviders `json:"providers"`
}

type opencodeProviders struct {
	Anthropic  opencodeProvider `json:"anthropic"`
	OpenAI     opencodeProvider `json:"openai"`
	OpenRouter opencodeProvider `json:"openrouter"`
}

type opencodeProvider struct {
	Enabled         bool               `json:"enabled"`
	Model           string             `json:"model,omitempty"`
	APIKeySecretRef *opencodeSecretRef `json:"apiKeySecretRef,omitempty"`
}

type opencodeSecretRef struct {
	Name      string `json:"name,omitempty"`
	Namespace string `json:"namespace,omitempty"`
}

func (r *OpenCodeWorkspaceReconciler) reconcileConfigMap(ctx context.Context, workspace *opencodev1alpha1.OpenCodeWorkspace, namespaceName string) error {
	config := opencodeConfig{
		Schema: "https://opencode.ai/config.json",
		Providers: opencodeProviders{
			Anthropic:  providerConfig(workspace.Spec.Providers.Anthropic.Enabled, workspace.Spec.Providers.Anthropic.Model, workspace.Spec.Providers.Anthropic.APIKeySecretRef),
			OpenAI:     providerConfig(workspace.Spec.Providers.OpenAI.Enabled, workspace.Spec.Providers.OpenAI.Model, workspace.Spec.Providers.OpenAI.APIKeySecretRef),
			OpenRouter: providerConfig(workspace.Spec.Providers.OpenRouter.Enabled, workspace.Spec.Providers.OpenRouter.Model, workspace.Spec.Providers.OpenRouter.APIKeySecretRef),
		},
	}

	payload, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal opencode config: %w", err)
	}

	configMap := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: opencodeConfigMapName, Namespace: namespaceName}}
	_, err = controllerutil.CreateOrUpdate(ctx, r.Client, configMap, func() error {
		if err := controllerutil.SetOwnerReference(workspace, configMap, r.Scheme); err != nil {
			return err
		}

		if configMap.Data == nil {
			configMap.Data = map[string]string{}
		}
		configMap.Data["opencode.jsonc"] = string(payload)
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to reconcile configmap %q: %w", opencodeConfigMapName, err)
	}

	return nil
}

func providerConfig(enabled bool, model string, secretRef opencodev1alpha1.SecretKeyRef) opencodeProvider {
	return opencodeProvider{
		Enabled:         enabled,
		Model:           model,
		APIKeySecretRef: convertSecretRef(secretRef),
	}
}

func convertSecretRef(secretRef opencodev1alpha1.SecretKeyRef) *opencodeSecretRef {
	if secretRef.Name == "" && secretRef.Namespace == "" {
		return nil
	}
	return &opencodeSecretRef{Name: secretRef.Name, Namespace: secretRef.Namespace}
}
