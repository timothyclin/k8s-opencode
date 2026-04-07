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

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	opencodev1alpha1 "github.com/timothyclin/k8s-opencode/operator/api/v1alpha1"
)

const (
	kubedockImage           = "joyrex2001/kubedock:0.20.3"
	kubedockPort            = 2475
	kubedockServiceName     = "kubedock-service"
	kubedockDeploymentName  = "kubedock"
	kubedockServiceAccount  = "kubedock"
	kubedockRoleName        = "kubedock"
	kubedockRoleBindingName = "kubedock"
)

func (r *OpenCodeWorkspaceReconciler) reconcileKubedock(ctx context.Context, workspace *opencodev1alpha1.OpenCodeWorkspace, namespaceName string) error {
	if !workspace.Spec.Kubedock.Enabled {
		return nil
	}

	// Create ServiceAccount
	if err := r.reconcileKubedockServiceAccount(ctx, workspace, namespaceName); err != nil {
		return err
	}

	// Create Role
	if err := r.reconcileKubedockRole(ctx, workspace, namespaceName); err != nil {
		return err
	}

	// Create RoleBinding
	if err := r.reconcileKubedockRoleBinding(ctx, workspace, namespaceName); err != nil {
		return err
	}

	// Create Deployment
	if err := r.reconcileKubedockDeployment(ctx, workspace, namespaceName); err != nil {
		return err
	}

	// Create Service
	if err := r.reconcileKubedockService(ctx, workspace, namespaceName); err != nil {
		return err
	}

	return nil
}

func (r *OpenCodeWorkspaceReconciler) reconcileKubedockServiceAccount(ctx context.Context, workspace *opencodev1alpha1.OpenCodeWorkspace, namespaceName string) error {
	sa := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      kubedockServiceAccount,
			Namespace: namespaceName,
		},
	}

	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, sa, func() error {
		if sa.Labels == nil {
			sa.Labels = map[string]string{}
		}
		sa.Labels["app.kubernetes.io/component"] = "kubedock"
		sa.Labels["opencode.io/workspace"] = workspace.Name

		if err := controllerutil.SetOwnerReference(workspace, sa, r.Scheme); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to reconcile kubedock ServiceAccount: %w", err)
	}
	return nil
}

func (r *OpenCodeWorkspaceReconciler) reconcileKubedockRole(ctx context.Context, workspace *opencodev1alpha1.OpenCodeWorkspace, namespaceName string) error {
	role := &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      kubedockRoleName,
			Namespace: namespaceName,
		},
	}

	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, role, func() error {
		if role.Labels == nil {
			role.Labels = map[string]string{}
		}
		role.Labels["app.kubernetes.io/component"] = "kubedock"
		role.Labels["opencode.io/workspace"] = workspace.Name

		role.Rules = []rbacv1.PolicyRule{
			{
				APIGroups: []string{""},
				Resources: []string{"pods", "services", "configmaps", "pods/exec", "pods/log"},
				Verbs:     []string{"get", "list", "watch", "create", "delete", "update", "patch"},
			},
			{
				APIGroups: []string{""},
				Resources: []string{"persistentvolumeclaims"},
				Verbs:     []string{"get", "list", "watch", "create", "delete"},
			},
		}

		if err := controllerutil.SetOwnerReference(workspace, role, r.Scheme); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to reconcile kubedock Role: %w", err)
	}
	return nil
}

func (r *OpenCodeWorkspaceReconciler) reconcileKubedockRoleBinding(ctx context.Context, workspace *opencodev1alpha1.OpenCodeWorkspace, namespaceName string) error {
	rb := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      kubedockRoleBindingName,
			Namespace: namespaceName,
		},
	}

	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, rb, func() error {
		if rb.Labels == nil {
			rb.Labels = map[string]string{}
		}
		rb.Labels["app.kubernetes.io/component"] = "kubedock"
		rb.Labels["opencode.io/workspace"] = workspace.Name

		rb.Subjects = []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      kubedockServiceAccount,
				Namespace: namespaceName,
			},
		}
		rb.RoleRef = rbacv1.RoleRef{
			Kind:     "Role",
			Name:     kubedockRoleName,
			APIGroup: "rbac.authorization.k8s.io",
		}

		if err := controllerutil.SetOwnerReference(workspace, rb, r.Scheme); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to reconcile kubedock RoleBinding: %w", err)
	}
	return nil
}

func (r *OpenCodeWorkspaceReconciler) reconcileKubedockDeployment(ctx context.Context, workspace *opencodev1alpha1.OpenCodeWorkspace, namespaceName string) error {
	labels := map[string]string{
		"app.kubernetes.io/component": "kubedock",
		"opencode.io/workspace":       workspace.Name,
	}
	replicas := int32(1)

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      kubedockDeploymentName,
			Namespace: namespaceName,
		},
	}

	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, deployment, func() error {
		if deployment.Labels == nil {
			deployment.Labels = map[string]string{}
		}
		for k, v := range labels {
			deployment.Labels[k] = v
		}

		deployment.Spec.Replicas = &replicas
		deployment.Spec.Selector = &metav1.LabelSelector{MatchLabels: labels}
		deployment.Spec.Template = corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{Labels: labels},
			Spec: corev1.PodSpec{
				ServiceAccountName: kubedockServiceAccount,
				Containers: []corev1.Container{
					{
						Name:            "kubedock",
						Image:           kubedockImage,
						ImagePullPolicy: corev1.PullIfNotPresent,
						Command:         []string{"kubedock", "server"},
						Args: []string{
							fmt.Sprintf("--listen-addr=:%d", kubedockPort),
							fmt.Sprintf("--labels=opencode.io/workspace=%s", workspace.Name),
						},
						Ports: []corev1.ContainerPort{
							{
								Name:          "api",
								ContainerPort: kubedockPort,
								Protocol:      corev1.ProtocolTCP,
							},
						},
						Resources: corev1.ResourceRequirements{
							Requests: workspace.Spec.Kubedock.Resources.Requests,
							Limits:   workspace.Spec.Kubedock.Resources.Limits,
						},
					},
				},
			},
		}

		// Apply default resources if not specified
		if deployment.Spec.Template.Spec.Containers[0].Resources.Requests == nil {
			deployment.Spec.Template.Spec.Containers[0].Resources.Requests = corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("50m"),
				corev1.ResourceMemory: resource.MustParse("64Mi"),
			}
		}
		if deployment.Spec.Template.Spec.Containers[0].Resources.Limits == nil {
			deployment.Spec.Template.Spec.Containers[0].Resources.Limits = corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("200m"),
				corev1.ResourceMemory: resource.MustParse("128Mi"),
			}
		}

		if err := controllerutil.SetOwnerReference(workspace, deployment, r.Scheme); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to reconcile kubedock Deployment: %w", err)
	}
	return nil
}

func (r *OpenCodeWorkspaceReconciler) reconcileKubedockService(ctx context.Context, workspace *opencodev1alpha1.OpenCodeWorkspace, namespaceName string) error {
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      kubedockServiceName,
			Namespace: namespaceName,
		},
	}

	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, service, func() error {
		if service.Labels == nil {
			service.Labels = map[string]string{}
		}
		service.Labels["app.kubernetes.io/component"] = "kubedock"
		service.Labels["opencode.io/workspace"] = workspace.Name

		service.Spec.Type = corev1.ServiceTypeClusterIP
		service.Spec.Selector = map[string]string{
			"app.kubernetes.io/component": "kubedock",
			"opencode.io/workspace":       workspace.Name,
		}
		service.Spec.Ports = []corev1.ServicePort{
			{
				Name:       "api",
				Protocol:   corev1.ProtocolTCP,
				Port:       kubedockPort,
				TargetPort: intstr.FromInt(kubedockPort),
			},
		}

		if err := controllerutil.SetOwnerReference(workspace, service, r.Scheme); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to reconcile kubedock Service: %w", err)
	}
	return nil
}
