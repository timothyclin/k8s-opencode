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
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	opencodev1alpha1 "github.com/timothyclin/k8s-opencode/operator/api/v1alpha1"
)

var _ = Describe("ConfigMap Reconciliation", func() {
	Context("When reconciling a workspace with MCP and plugins", func() {
		It("Should generate complete opencode.json with plugins, MCPs, and skills", func() {
			ctx := context.Background()

			workspace := &opencodev1alpha1.OpenCodeWorkspace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-workspace",
				},
				Spec: opencodev1alpha1.OpenCodeWorkspaceSpec{
					Email:           "test@example.com",
					NamespacePrefix: "oc",
					Plugins: opencodev1alpha1.PluginsSpec{
						Enabled: true,
						NPM: []string{
							"superpowers@git+https://github.com/obra/superpowers.git",
						},
					},
					MCP: opencodev1alpha1.MCPSpec{
						Remote: []opencodev1alpha1.RemoteMCPServer{
							{
								Name:    "context7",
								URL:     "https://mcp.context7.com/mcp",
								Enabled: true,
							},
						},
					},
					Skills: opencodev1alpha1.SkillsSpec{
						NPM: []string{"my-skill-package@latest"},
					},
				},
			}

			// Create namespace
			ns := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "oc-test-workspace",
				},
			}
			Expect(k8sClient.Create(ctx, ns)).To(Succeed())

			// Create workspace
			Expect(k8sClient.Create(ctx, workspace)).To(Succeed())

			// Reconcile ConfigMap
			reconciler := &OpenCodeWorkspaceReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			apiKeys := map[string]string{}
			err := reconciler.reconcileConfigMap(ctx, workspace, "oc-test-workspace", apiKeys)
			Expect(err).NotTo(HaveOccurred())

			// Verify ConfigMap was created
			configMap := &corev1.ConfigMap{}
			err = k8sClient.Get(ctx, types.NamespacedName{
				Name:      "opencode-config",
				Namespace: "oc-test-workspace",
			}, configMap)
			Expect(err).NotTo(HaveOccurred())

			// Parse and verify the JSON content
			var config opencodeConfig
			err = json.Unmarshal([]byte(configMap.Data["opencode.json"]), &config)
			Expect(err).NotTo(HaveOccurred())

			// Verify schema
			Expect(config.Schema).To(Equal("https://opencode.ai/config.json"))

			// Verify plugins
			Expect(config.Plugin).To(ContainElements(
				"oh-my-opencode@latest",
				"@tarquinen/opencode-dcp@latest",
				"superpowers@git+https://github.com/obra/superpowers.git",
			))

			// Verify MCP servers
			Expect(config.MCP).To(HaveKey("context7"))
			Expect(config.MCP["context7"].Type).To(Equal("remote"))
			Expect(config.MCP["context7"].URL).To(Equal("https://mcp.context7.com/mcp"))
			Expect(config.MCP["context7"].Enabled).To(BeTrue())

			// Verify skills
			Expect(config.Skills).NotTo(BeNil())
			Expect(config.Skills.NPM).To(ContainElement("my-skill-package@latest"))
		})
	})
})

func TestReconcileConfigMap(t *testing.T) {
	// This is a placeholder for go test discovery
	// Actual tests run via Ginkgo
}
