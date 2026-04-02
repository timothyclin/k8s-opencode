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

// opencodeConfig matches the anomalyco/opencode config schema
// (NOT opencode-ai/opencode which uses "providers" plural)
type opencodeConfig struct {
	Schema           string                      `json:"$schema,omitempty"`
	EnabledProviders []string                    `json:"enabled_providers,omitempty"`
	Provider         map[string]*providerOptions `json:"provider,omitempty"`
}

// providerOptions holds per-provider configuration
type providerOptions struct {
	Options *providerOptionsInner `json:"options,omitempty"`
}

type providerOptionsInner struct {
	APIKey string `json:"apiKey,omitempty"`
}

func (r *OpenCodeWorkspaceReconciler) reconcileConfigMap(ctx context.Context, workspace *opencodev1alpha1.OpenCodeWorkspace, namespaceName string, apiKeys map[string]string) error {
	var enabledProviders []string
	provider := make(map[string]*providerOptions)

	if workspace.Spec.Providers.Anthropic.Enabled {
		enabledProviders = append(enabledProviders, "anthropic")
		if key := apiKeys["anthropic"]; key != "" {
			provider["anthropic"] = &providerOptions{Options: &providerOptionsInner{APIKey: key}}
		}
	}

	if workspace.Spec.Providers.OpenAI.Enabled {
		enabledProviders = append(enabledProviders, "openai")
		if key := apiKeys["openai"]; key != "" {
			provider["openai"] = &providerOptions{Options: &providerOptionsInner{APIKey: key}}
		}
	}

	if workspace.Spec.Providers.OpenRouter.Enabled {
		enabledProviders = append(enabledProviders, "openrouter")
		if key := apiKeys["openrouter"]; key != "" {
			provider["openrouter"] = &providerOptions{Options: &providerOptionsInner{APIKey: key}}
		}
	}

	config := opencodeConfig{
		Schema:           "https://opencode.ai/config.json",
		EnabledProviders: enabledProviders,
	}

	if len(provider) > 0 {
		config.Provider = provider
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
		configMap.Data["opencode.json"] = string(payload)
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to reconcile configmap %q: %w", opencodeConfigMapName, err)
	}

	return nil
}
