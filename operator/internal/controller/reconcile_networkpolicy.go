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

	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	opencodev1alpha1 "github.com/timothyclin/k8s-opencode/operator/api/v1alpha1"
)

const networkPolicyName = "workspace-isolation"

// reconcileNetworkPolicy creates a NetworkPolicy that isolates the workspace namespace.
// It allows:
// - Egress to anywhere (workspace needs to call LLM APIs, git, etc.)
// - Ingress only from the Tailscale ingress controller namespace
func (r *OpenCodeWorkspaceReconciler) reconcileNetworkPolicy(ctx context.Context, workspace *opencodev1alpha1.OpenCodeWorkspace, namespaceName string) error {
	np := &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      networkPolicyName,
			Namespace: namespaceName,
		},
	}

	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, np, func() error {
		if err := controllerutil.SetOwnerReference(workspace, np, r.Scheme); err != nil {
			return err
		}

		// Select all pods in namespace
		np.Spec.PodSelector = metav1.LabelSelector{}

		// Policy types: both ingress and egress
		np.Spec.PolicyTypes = []networkingv1.PolicyType{
			networkingv1.PolicyTypeIngress,
			networkingv1.PolicyTypeEgress,
		}

		// Ingress: allow from tailscale namespace (ingress controller)
		np.Spec.Ingress = []networkingv1.NetworkPolicyIngressRule{
			{
				From: []networkingv1.NetworkPolicyPeer{
					{
						NamespaceSelector: &metav1.LabelSelector{
							MatchLabels: map[string]string{
								"kubernetes.io/metadata.name": "tailscale",
							},
						},
					},
				},
			},
		}

		// Egress: allow all (workspace needs external access)
		np.Spec.Egress = []networkingv1.NetworkPolicyEgressRule{
			{}, // Empty rule = allow all
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to reconcile network policy %q: %w", networkPolicyName, err)
	}

	return nil
}
