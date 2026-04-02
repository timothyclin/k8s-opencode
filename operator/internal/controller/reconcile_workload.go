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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	opencodev1alpha1 "github.com/timothyclin/k8s-opencode/operator/api/v1alpha1"
)

const (
	workspaceContainerImage = "ghcr.io/timothyclin/k8s-opencode/opencode-workspace:latest"
	workspaceServiceName    = "workspace"
	workspaceConfigMapName  = "opencode-config"
	workspaceDataPVCName    = "data-pvc"
	workspacePVCName        = "workspace-pvc"
)

func (r *OpenCodeWorkspaceReconciler) reconcileStatefulSet(ctx context.Context, workspace *opencodev1alpha1.OpenCodeWorkspace, namespaceName string) error {
	labels := map[string]string{
		"app":                   "workspace",
		"opencode.io/workspace": workspace.Name,
	}
	replicas := int32(1)

	statefulSet := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      workspaceServiceName,
			Namespace: namespaceName,
		},
	}

	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, statefulSet, func() error {
		if statefulSet.Labels == nil {
			statefulSet.Labels = map[string]string{}
		}
		statefulSet.Labels["app"] = labels["app"]
		statefulSet.Labels["opencode.io/workspace"] = labels["opencode.io/workspace"]

		statefulSet.Spec.Replicas = &replicas
		statefulSet.Spec.ServiceName = workspaceServiceName
		statefulSet.Spec.Selector = &metav1.LabelSelector{MatchLabels: labels}
		statefulSet.Spec.Template = corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{Labels: labels},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:      "workspace",
						Image:     workspaceContainerImage,
						Resources: workspace.Spec.Resources,
						Ports: []corev1.ContainerPort{
							{
								Name:          "http",
								ContainerPort: 4096,
							},
						},
						Env: buildProviderEnvVars(workspace),
						VolumeMounts: []corev1.VolumeMount{
							{
								Name:      workspacePVCName,
								MountPath: "/workspace",
							},
							{
								Name:      workspaceDataPVCName,
								MountPath: "/home/opencode/.opencode",
							},
							{
								Name:      workspaceConfigMapName,
								MountPath: "/home/opencode/.opencode/opencode.json",
								SubPath:   "opencode.json",
								ReadOnly:  true,
							},
						},
					},
				},
				Volumes: []corev1.Volume{
					{
						Name: workspacePVCName,
						VolumeSource: corev1.VolumeSource{
							PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
								ClaimName: workspacePVCName,
							},
						},
					},
					{
						Name: workspaceDataPVCName,
						VolumeSource: corev1.VolumeSource{
							PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
								ClaimName: workspaceDataPVCName,
							},
						},
					},
					{
						Name: workspaceConfigMapName,
						VolumeSource: corev1.VolumeSource{
							ConfigMap: &corev1.ConfigMapVolumeSource{
								LocalObjectReference: corev1.LocalObjectReference{Name: workspaceConfigMapName},
							},
						},
					},
				},
			},
		}

		if err := controllerutil.SetOwnerReference(workspace, statefulSet, r.Scheme); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to reconcile statefulset %q: %w", statefulSet.Name, err)
	}
	return nil
}

func (r *OpenCodeWorkspaceReconciler) reconcileService(ctx context.Context, workspace *opencodev1alpha1.OpenCodeWorkspace, namespaceName string) error {
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      workspaceServiceName,
			Namespace: namespaceName,
		},
	}

	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, service, func() error {
		if service.Labels == nil {
			service.Labels = map[string]string{}
		}
		service.Labels["app"] = "workspace"
		service.Labels["opencode.io/workspace"] = workspace.Name

		service.Spec.Type = corev1.ServiceTypeClusterIP
		service.Spec.Selector = map[string]string{"app": "workspace"}
		service.Spec.Ports = []corev1.ServicePort{
			{
				Name:       "http",
				Port:       4096,
				TargetPort: intstr.FromInt(4096),
			},
		}

		if err := controllerutil.SetOwnerReference(workspace, service, r.Scheme); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to reconcile service %q: %w", service.Name, err)
	}
	return nil
}

func buildProviderEnvVars(workspace *opencodev1alpha1.OpenCodeWorkspace) []corev1.EnvVar {
	var envs []corev1.EnvVar
	if workspace.Spec.Providers.Anthropic.Enabled && workspace.Spec.Providers.Anthropic.APIKeySecretRef.Name != "" {
		envs = append(envs, secretEnvVar("ANTHROPIC_API_KEY", workspace.Spec.Providers.Anthropic.APIKeySecretRef.Name))
	}
	if workspace.Spec.Providers.OpenAI.Enabled && workspace.Spec.Providers.OpenAI.APIKeySecretRef.Name != "" {
		envs = append(envs, secretEnvVar("OPENAI_API_KEY", workspace.Spec.Providers.OpenAI.APIKeySecretRef.Name))
	}
	if workspace.Spec.Providers.OpenRouter.Enabled && workspace.Spec.Providers.OpenRouter.APIKeySecretRef.Name != "" {
		envs = append(envs, secretEnvVar("OPENROUTER_API_KEY", workspace.Spec.Providers.OpenRouter.APIKeySecretRef.Name))
	}
	return envs
}

func secretEnvVar(name, secretName string) corev1.EnvVar {
	return corev1.EnvVar{
		Name: name,
		ValueFrom: &corev1.EnvVarSource{
			SecretKeyRef: &corev1.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{Name: secretName},
				Key:                  "api-key",
			},
		},
	}
}
