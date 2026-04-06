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
	Plugin           []string                    `json:"plugin,omitempty"`
	EnabledProviders []string                    `json:"enabled_providers,omitempty"`
	Provider         map[string]*providerOptions `json:"provider,omitempty"`
	MCP              map[string]*mcpServer       `json:"mcp,omitempty"`
	Skills           *skillsConfig               `json:"skills,omitempty"`
}

// providerOptions holds per-provider configuration
type providerOptions struct {
	Options *providerOptionsInner `json:"options,omitempty"`
}

type providerOptionsInner struct {
	APIKey string `json:"apiKey,omitempty"`
}

// mcpServer defines an MCP server entry in opencode.json
type mcpServer struct {
	Type    string            `json:"type"`
	URL     string            `json:"url"`
	Enabled bool              `json:"enabled,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
	OAuth   *mcpOAuth         `json:"oauth,omitempty"`
}

// mcpOAuth defines OAuth configuration for MCP servers
type mcpOAuth struct {
	ClientID     string `json:"clientId,omitempty"`
	ClientSecret string `json:"clientSecret,omitempty"`
	TokenURL     string `json:"tokenUrl,omitempty"`
}

// skillsConfig defines skills configuration
type skillsConfig struct {
	NPM    []string `json:"npm,omitempty"`
	Config []any    `json:"config,omitempty"`
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

	// Add plugins
	if workspace.Spec.Plugins.Enabled {
		config.Plugin = []string{
			"oh-my-opencode@latest",
			"@tarquinen/opencode-dcp@latest",
		}
		// Add user-specified plugins
		config.Plugin = append(config.Plugin, workspace.Spec.Plugins.NPM...)
	}

	// Add MCP servers
	mcpServers := make(map[string]*mcpServer)
	for _, remote := range workspace.Spec.MCP.Remote {
		srv := &mcpServer{
			Type:    "remote",
			URL:     remote.URL,
			Enabled: remote.Enabled,
		}
		if len(remote.Headers) > 0 {
			srv.Headers = remote.Headers
		}
		if remote.OAuth != nil {
			srv.OAuth = &mcpOAuth{
				ClientID:     remote.OAuth.ClientID,
				ClientSecret: remote.OAuth.ClientSecret,
				TokenURL:     remote.OAuth.TokenURL,
			}
		}
		mcpServers[remote.Name] = srv
	}

	// Add laptop MCP servers
	for _, laptop := range workspace.Spec.MCP.LaptopServers {
		host := laptop.TailscaleIP
		if laptop.TailscaleFQDN != "" {
			host = laptop.TailscaleFQDN
		}
		mcpServers["laptop_"+laptop.Name] = &mcpServer{
			Type:    "remote",
			URL:     fmt.Sprintf("http://%s:%d", host, laptop.Port),
			Enabled: laptop.Enabled,
		}
	}

	if len(mcpServers) > 0 {
		config.MCP = mcpServers
	}

	// Add skills
	if len(workspace.Spec.Skills.NPM) > 0 || len(workspace.Spec.Skills.Config) > 0 {
		skillsCfg := &skillsConfig{
			NPM: workspace.Spec.Skills.NPM,
		}
		// Convert runtime.RawExtension to any for JSON marshaling
		for _, raw := range workspace.Spec.Skills.Config {
			var obj any
			if err := json.Unmarshal(raw.Raw, &obj); err != nil {
				return fmt.Errorf("unmarshal skill config: %w", err)
			}
			skillsCfg.Config = append(skillsCfg.Config, obj)
		}
		config.Skills = skillsCfg
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
