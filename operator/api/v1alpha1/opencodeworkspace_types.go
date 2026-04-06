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

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// OpenCodeWorkspaceSpec defines the desired state of OpenCodeWorkspace
type OpenCodeWorkspaceSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	// The following markers will use OpenAPI v3 schema to validate the value
	// More info: https://book.kubebuilder.io/reference/markers/crd-validation.html

	// Email is the identity email address for the workspace owner.
	//
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern=`^[^@]+@[^@]+$`
	Email string `json:"email"`

	// NamespacePrefix is the prefix for the workspace namespace.
	// If username is "timothy" and prefix is "oc", namespace will be "oc-timothy".
	// Defaults to "oc".
	//
	// +optional
	// +kubebuilder:default="oc"
	NamespacePrefix string `json:"namespacePrefix,omitempty"`

	// Providers configures one or more AI providers.
	// +optional
	Providers ProvidersSpec `json:"providers,omitempty"`

	// Resources defines CPU/memory requests and limits for the workspace workload.
	// +optional
	Resources ResourceRequirements `json:"resources,omitempty"`

	// Storage defines persistent volume sizes for the workspace.
	// +optional
	Storage StorageSpec `json:"storage,omitempty"`

	// Tailscale configures tailnet ingress and optional egress connectivity.
	// +optional
	Tailscale TailscaleSpec `json:"tailscale,omitempty"`

	// Kubedock configures the optional kubedock sidecar.
	// +optional
	Kubedock KubedockSpec `json:"kubedock,omitempty"`

	// ServerPassword is the password for the OpenCode HTTP server.
	// If not provided, a random password will be generated.
	// The password is stored in the workspace-secrets Secret.
	// +optional
	ServerPassword string `json:"serverPassword,omitempty"`

	// Plugins configures npm plugins to install in the workspace.
	// Default plugins (oh-my-opencode@latest, @tarquinen/opencode-dcp@latest, superpowers)
	// are always included unless explicitly disabled.
	// +optional
	Plugins PluginsSpec `json:"plugins,omitempty"`

	// MCP configures Model Context Protocol servers.
	// +optional
	MCP MCPSpec `json:"mcp,omitempty"`

	// Skills configures skills (reusable prompt templates and tool configurations).
	// +optional
	Skills SkillsSpec `json:"skills,omitempty"`
}

// ProvidersSpec configures supported AI providers.
type ProvidersSpec struct {
	// Anthropic config.
	// +optional
	Anthropic AnthropicProviderConfig `json:"anthropic,omitempty"`

	// OpenAI config.
	// +optional
	OpenAI ProviderConfig `json:"openai,omitempty"`

	// OpenRouter config.
	// +optional
	OpenRouter ProviderConfig `json:"openrouter,omitempty"`
}

// ProviderConfig defines configuration for an AI provider.
type ProviderConfig struct {
	// Enabled toggles whether this provider is enabled for the workspace.
	// +optional
	Enabled bool `json:"enabled,omitempty"`

	// Model is the provider-specific model identifier.
	// +optional
	Model string `json:"model,omitempty"`

	// APIKeySecretRef references a Secret containing the provider API key.
	// +optional
	APIKeySecretRef SecretKeyRef `json:"apiKeySecretRef,omitempty"`
}

// AnthropicProviderConfig defines configuration for the Anthropic provider.
type AnthropicProviderConfig struct {
	// Enabled toggles whether this provider is enabled for the workspace.
	// +optional
	Enabled bool `json:"enabled,omitempty"`

	// Model is the Anthropic model identifier.
	// +optional
	// +kubebuilder:default="claude-sonnet-4-20250514"
	Model string `json:"model,omitempty"`

	// APIKeySecretRef references a Secret containing the provider API key.
	// +optional
	APIKeySecretRef SecretKeyRef `json:"apiKeySecretRef,omitempty"`
}

// SecretKeyRef identifies a Secret by name/namespace.
//
// Since OpenCodeWorkspace is cluster-scoped, this reference must include a namespace.
type SecretKeyRef struct {
	// Name is the Secret name.
	// +optional
	Name string `json:"name,omitempty"`

	// Namespace is the Secret namespace.
	// +optional
	Namespace string `json:"namespace,omitempty"`
}

// ResourceRequirements is an alias for corev1.ResourceRequirements.
//
// This is the standard Kubernetes shape for requests/limits.
type ResourceRequirements = corev1.ResourceRequirements

// StorageSpec defines persistent storage settings.
type StorageSpec struct {
	// Workspace is the PVC size for /workspace.
	// +optional
	Workspace resource.Quantity `json:"workspace,omitempty"`

	// Data is the PVC size for ~/.opencode.
	// +optional
	Data resource.Quantity `json:"data,omitempty"`

	// StorageClassName is the StorageClass used for PVCs. Empty means the cluster default.
	// +optional
	StorageClassName string `json:"storageClassName,omitempty"`
}

// TailscaleSpec defines tailnet connectivity settings.
type TailscaleSpec struct {
	// IngressTags are tags applied to the Tailscale ingress identity.
	// +optional
	IngressTags []string `json:"ingressTags,omitempty"`

	// Egress configures optional egress connectivity (e.g., to laptop MCP servers).
	// +optional
	Egress EgressSpec `json:"egress,omitempty"`
}

// EgressSpec defines egress proxy configuration.
type EgressSpec struct {
	// Enabled turns on egress connectivity.
	// +optional
	Enabled bool `json:"enabled,omitempty"`

	// LaptopHostname is the MagicDNS hostname for the user's laptop.
	// +optional
	LaptopHostname string `json:"laptopHostname,omitempty"`

	// MCPPorts lists MCP services exposed from the laptop via ExternalName Services.
	// +optional
	MCPPorts []MCPPort `json:"mcpPorts,omitempty"`

	// Tags are tags applied to the egress proxy identity.
	// +optional
	Tags []string `json:"tags,omitempty"`
}

// MCPPort defines a single MCP server port.
type MCPPort struct {
	// Name is a short identifier for the MCP server.
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`

	// Port is the TCP port exposed by the MCP server.
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=65535
	Port int32 `json:"port"`
}

// KubedockSpec configures the optional kubedock sidecar.
type KubedockSpec struct {
	// Enabled turns on the kubedock sidecar.
	// +optional
	Enabled bool `json:"enabled,omitempty"`

	// Resources defines CPU/memory requests and limits for kubedock.
	// +optional
	Resources ResourceRequirements `json:"resources,omitempty"`
}

// PluginsSpec configures npm plugins for OpenCode.
type PluginsSpec struct {
	// Enabled controls whether default plugins are loaded.
	// Default plugins: oh-my-opencode@latest, @tarquinen/opencode-dcp@latest, superpowers
	// +optional
	// +kubebuilder:default=true
	Enabled bool `json:"enabled,omitempty"`

	// NPM is a list of additional npm packages to install.
	// Example: ["my-plugin@latest", "@scope/plugin@1.0.0"]
	// +optional
	NPM []string `json:"npm,omitempty"`
}

// MCPSpec configures Model Context Protocol servers.
type MCPSpec struct {
	// Remote lists remote MCP servers accessible via HTTP.
	// +optional
	Remote []RemoteMCPServer `json:"remote,omitempty"`

	// LaptopServers lists MCP servers running on the user's laptop (via Tailscale egress).
	// +optional
	LaptopServers []LaptopMCPServer `json:"laptopServers,omitempty"`
}

// RemoteMCPServer defines a remote MCP server configuration.
type RemoteMCPServer struct {
	// Name is a unique identifier for this MCP server.
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`

	// URL is the HTTP endpoint for the MCP server.
	// +kubebuilder:validation:MinLength=1
	URL string `json:"url"`

	// Enabled controls whether this MCP server is active.
	// +optional
	// +kubebuilder:default=true
	Enabled bool `json:"enabled,omitempty"`

	// Headers are optional HTTP headers to include in MCP requests.
	// +optional
	Headers map[string]string `json:"headers,omitempty"`

	// OAuth configures optional OAuth2 authentication.
	// +optional
	OAuth *MCPOAuthConfig `json:"oauth,omitempty"`
}

// LaptopMCPServer defines an MCP server running on the user's laptop.
type LaptopMCPServer struct {
	// Name is a unique identifier for this MCP server.
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`

	// TailscaleIP is the Tailscale IP address of the laptop.
	// +optional
	TailscaleIP string `json:"tailscaleIP,omitempty"`

	// TailscaleFQDN is the MagicDNS hostname of the laptop.
	// +optional
	TailscaleFQDN string `json:"tailscaleFqdn,omitempty"`

	// Port is the TCP port where the MCP server listens.
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=65535
	Port int32 `json:"port"`

	// Enabled controls whether this MCP server is active.
	// +optional
	// +kubebuilder:default=true
	Enabled bool `json:"enabled,omitempty"`
}

// MCPOAuthConfig defines OAuth2 authentication for MCP servers.
type MCPOAuthConfig struct {
	// ClientID is the OAuth2 client ID.
	// +optional
	ClientID string `json:"clientId,omitempty"`

	// ClientSecret is the OAuth2 client secret.
	// +optional
	ClientSecret string `json:"clientSecret,omitempty"`

	// TokenURL is the OAuth2 token endpoint.
	// +optional
	TokenURL string `json:"tokenUrl,omitempty"`
}

// SkillsSpec configures skills (reusable prompt templates and tool configurations).
type SkillsSpec struct {
	// NPM is a list of npm packages providing skills.
	// +optional
	NPM []string `json:"npm,omitempty"`

	// Config is a list of inline skill configurations as raw JSON.
	// +optional
	// +kubebuilder:pruning:PreserveUnknownFields
	// +kubebuilder:validation:Schemaless
	Config []runtime.RawExtension `json:"config,omitempty"`
}

// OpenCodeWorkspaceStatus defines the observed state of OpenCodeWorkspace.
type OpenCodeWorkspaceStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// For Kubernetes API conventions, see:
	// https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#typical-status-properties

	// Phase is a high-level summary of the workspace lifecycle.
	//
	// +kubebuilder:validation:Enum=Pending;Creating;Reconciling;Running;Failed;Terminating
	// +optional
	Phase string `json:"phase,omitempty"`

	// Namespace is the namespace created/managed for this workspace.
	// +optional
	Namespace string `json:"namespace,omitempty"`

	// IngressHostname is the tailnet hostname used to reach the workspace.
	// +optional
	IngressHostname string `json:"ingressHostname,omitempty"`

	// Message is a human-readable status message.
	// +optional
	Message string `json:"message,omitempty"`

	// ACLSnippet is a recommended ACL policy snippet for tailnet configuration.
	// +optional
	ACLSnippet string `json:"aclSnippet,omitempty"`

	// conditions represent the current state of the OpenCodeWorkspace resource.
	// Each condition has a unique type and reflects the status of a specific aspect of the resource.
	//
	// Standard condition types include:
	// - "Available": the resource is fully functional
	// - "Progressing": the resource is being created or updated
	// - "Degraded": the resource failed to reach or maintain its desired state
	//
	// The status of each condition is one of True, False, or Unknown.
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster

// OpenCodeWorkspace is the Schema for the opencodeworkspaces API
type OpenCodeWorkspace struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitzero"`

	// spec defines the desired state of OpenCodeWorkspace
	// +required
	Spec OpenCodeWorkspaceSpec `json:"spec"`

	// status defines the observed state of OpenCodeWorkspace
	// +optional
	Status OpenCodeWorkspaceStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true

// OpenCodeWorkspaceList contains a list of OpenCodeWorkspace
type OpenCodeWorkspaceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []OpenCodeWorkspace `json:"items"`
}

func init() {
	SchemeBuilder.Register(&OpenCodeWorkspace{}, &OpenCodeWorkspaceList{})
}
