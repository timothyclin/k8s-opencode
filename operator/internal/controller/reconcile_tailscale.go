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

const (
	ingressClassName = "tailscale"
)

func (r *OpenCodeWorkspaceReconciler) reconcileTailscaleIngress(ctx context.Context, workspace *opencodev1alpha1.OpenCodeWorkspace, namespaceName string) error {
	if workspace.Spec.Tailscale.IngressTags == nil && !workspace.Spec.Tailscale.Egress.Enabled {
		return nil
	}

	hostname := generateIngressHostname(workspace)

	if err := r.reconcileIngress(ctx, workspace, namespaceName, hostname); err != nil {
		return fmt.Errorf("failed to reconcile Ingress: %w", err)
	}

	if workspace.Status.IngressHostname != hostname {
		workspace.Status.IngressHostname = hostname
	}

	return nil
}

func (r *OpenCodeWorkspaceReconciler) reconcileIngress(ctx context.Context, workspace *opencodev1alpha1.OpenCodeWorkspace, namespaceName, hostname string) error {
	ingress := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "opencode-workspace",
			Namespace: namespaceName,
		},
	}

	className := ingressClassName

	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, ingress, func() error {
		if ingress.Labels == nil {
			ingress.Labels = map[string]string{}
		}
		ingress.Labels["app.kubernetes.io/managed-by"] = "opencode-operator"
		ingress.Labels["app.kubernetes.io/workspace"] = workspace.Name

		if ingress.Annotations == nil {
			ingress.Annotations = map[string]string{}
		}
		ingress.Annotations["tailscale.com/hostname"] = hostname

		if len(workspace.Spec.Tailscale.IngressTags) > 0 {
			ingress.Annotations["tailscale.com/tags"] = fmt.Sprintf("%s", workspace.Spec.Tailscale.IngressTags[0])
		}

		ingress.Spec = networkingv1.IngressSpec{
			IngressClassName: &className,
			DefaultBackend: &networkingv1.IngressBackend{
				Service: &networkingv1.IngressServiceBackend{
					Name: "workspace",
					Port: networkingv1.ServiceBackendPort{
						Number: 4096,
					},
				},
			},
			TLS: []networkingv1.IngressTLS{
				{
					Hosts: []string{hostname},
				},
			},
		}

		if err := controllerutil.SetOwnerReference(workspace, ingress, r.Scheme); err != nil {
			return err
		}
		return nil
	})
	return err
}

func generateIngressHostname(workspace *opencodev1alpha1.OpenCodeWorkspace) string {
	nsPrefix := workspace.Spec.NamespacePrefix
	if nsPrefix == "" {
		nsPrefix = "oc"
	}
	return fmt.Sprintf("oc-%s-%s", workspace.Name, nsPrefix)
}
