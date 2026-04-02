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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	opencodev1alpha1 "github.com/timothyclin/k8s-opencode/operator/api/v1alpha1"
)

const (
	namespacePrefix     = "opencode-"
	workspaceOwnerLabel = "opencode.io/workspace"
)

// reconcileNamespace ensures the workspace namespace exists.
// Returns the namespace name.
func (r *OpenCodeWorkspaceReconciler) reconcileNamespace(ctx context.Context, workspace *opencodev1alpha1.OpenCodeWorkspace) (string, error) {
	namespaceName := namespacePrefix + workspace.Name

	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespaceName,
		},
	}

	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, ns, func() error {
		// Set owner reference for garbage collection.
		// Note: For cluster-scoped owner -> namespace-scoped owned, we use SetOwnerReference
		// (not SetControllerReference which requires same-namespace).
		if err := controllerutil.SetOwnerReference(workspace, ns, r.Scheme); err != nil {
			return err
		}

		// Add labels for identification
		if ns.Labels == nil {
			ns.Labels = map[string]string{}
		}
		ns.Labels[workspaceOwnerLabel] = workspace.Name

		return nil
	})
	if err != nil {
		return "", fmt.Errorf("failed to reconcile namespace %q: %w", namespaceName, err)
	}

	return namespaceName, nil
}
