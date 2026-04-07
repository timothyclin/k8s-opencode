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
	"k8s.io/utils/ptr"
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
				SecurityContext: &corev1.PodSecurityContext{
					RunAsNonRoot: ptr.To(true),
					RunAsUser:    ptr.To(int64(1000)),
					RunAsGroup:   ptr.To(int64(1000)),
					FSGroup:      ptr.To(int64(1000)),
				},
				InitContainers: []corev1.Container{
					{
						Name:    "init-permissions",
						Image:   workspaceContainerImage,
						Command: []string{"sh", "-c"},
						Args: []string{`set -e
USER_NAME="${WORKSPACE_USER}"
USER_ID="1000"
USER_GID="1000"

# Create user with provided name
id "$USER_NAME" 2>/dev/null || useradd -u $USER_ID -g $USER_GID -m -d /home/$USER_NAME -s /bin/bash "$USER_NAME"

# Configure sudoers for the runtime user
echo "$USER_NAME ALL=(ALL) NOPASSWD:ALL" > /etc/sudoers.d/$USER_NAME
chmod 440 /etc/sudoers.d/$USER_NAME
# sudo refuses sudoers.d if the directory is world-writable (EmptyDir default is 0777)
chmod 750 /etc/sudoers.d

# Build /etc/passwd and /etc/shadow with dynamic user entry
cp /etc/passwd /mnt/passwd/passwd
cp /etc/shadow /mnt/shadow/shadow
grep -q "^$USER_NAME:" /mnt/shadow/shadow || echo "$USER_NAME:*:0:0:99999:7:::" >> /mnt/shadow/shadow
chmod 640 /mnt/shadow/shadow

      # Fix ownership on mounted directories
      chown -R $USER_ID:$USER_GID /home/$USER_NAME 2>/dev/null || true

# Copy config files from ConfigMap to writable location (ConfigMap is read-only)
if [ -f /etc/opencode-config/opencode.json ]; then
  mkdir -p /home/$USER_NAME/.opencode 2>/dev/null || true
  cp /etc/opencode-config/opencode.json /home/$USER_NAME/.opencode/
fi
`},
						SecurityContext: &corev1.SecurityContext{
							RunAsUser:    ptr.To(int64(0)),
							RunAsGroup:   ptr.To(int64(0)),
							RunAsNonRoot: ptr.To(false),
						},
						Env: []corev1.EnvVar{
							{
								Name:  "WORKSPACE_USER",
								Value: workspace.Name,
							},
						},
						VolumeMounts: []corev1.VolumeMount{
							{Name: "sudoers", MountPath: "/etc/sudoers.d"},
							{Name: "passwd", MountPath: "/mnt/passwd"},
							{Name: "shadow", MountPath: "/mnt/shadow"},
							{Name: workspaceDataPVCName, MountPath: fmt.Sprintf("/home/%s", workspace.Name)},
							{Name: workspacePVCName, MountPath: fmt.Sprintf("/home/%s/workspace", workspace.Name)},
							{Name: workspaceConfigMapName, MountPath: "/etc/opencode-config", ReadOnly: true},
						},
					},
				},
				Containers: []corev1.Container{
					{
						Name:       "workspace",
						Image:      workspaceContainerImage,
						Command:    []string{"opencode", "serve", "--hostname", "0.0.0.0", "--port", "4096"},
						WorkingDir: fmt.Sprintf("/home/%s", workspace.Name),
						Resources:  workspace.Spec.Resources,
						Ports: []corev1.ContainerPort{
							{
								Name:          "http",
								ContainerPort: 4096,
							},
						},
						SecurityContext: &corev1.SecurityContext{
							RunAsUser:    ptr.To(int64(1000)),
							RunAsGroup:   ptr.To(int64(1000)),
							RunAsNonRoot: ptr.To(true),
						},
						Env: buildProviderEnvVars(workspace),
						VolumeMounts: []corev1.VolumeMount{
							{
								Name:      workspacePVCName,
								MountPath: fmt.Sprintf("/home/%s/workspace", workspace.Name),
							},
							{
								Name:      workspaceDataPVCName,
								MountPath: fmt.Sprintf("/home/%s", workspace.Name),
							},
							{
								Name:      "sudoers",
								MountPath: "/etc/sudoers.d",
								ReadOnly:  true,
							},
							{
								Name:      "passwd",
								MountPath: "/etc/passwd",
								SubPath:   "passwd",
								ReadOnly:  true,
							},
							{
								Name:      "shadow",
								MountPath: "/etc/shadow",
								SubPath:   "shadow",
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
					{
						Name: "sudoers",
						VolumeSource: corev1.VolumeSource{
							EmptyDir: &corev1.EmptyDirVolumeSource{},
						},
					},
					{
						Name: "passwd",
						VolumeSource: corev1.VolumeSource{
							EmptyDir: &corev1.EmptyDirVolumeSource{},
						},
					},
					{
						Name: "shadow",
						VolumeSource: corev1.VolumeSource{
							EmptyDir: &corev1.EmptyDirVolumeSource{},
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

	// Set HOME environment variable to user's home directory
	envs = append(envs, corev1.EnvVar{
		Name:  "HOME",
		Value: fmt.Sprintf("/home/%s", workspace.Name),
	})

	// Always inject server password (required for OpenCode HTTP server)
	envs = append(envs, corev1.EnvVar{
		Name: "OPENCODE_SERVER_PASSWORD",
		ValueFrom: &corev1.EnvVarSource{
			SecretKeyRef: &corev1.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{Name: "workspace-secrets"},
				Key:                  "server-password",
			},
		},
	})

	// Inject kubedock DOCKER_HOST if enabled
	if workspace.Spec.Kubedock.Enabled {
		envs = append(envs, corev1.EnvVar{
			Name:  "DOCKER_HOST",
			Value: fmt.Sprintf("tcp://%s:%d", kubedockServiceName, kubedockPort),
		})
		envs = append(envs, corev1.EnvVar{
			Name:  "TESTCONTAINERS_RYUK_DISABLED",
			Value: "true",
		})
		envs = append(envs, corev1.EnvVar{
			Name:  "TESTCONTAINERS_CHECKS_DISABLE",
			Value: "true",
		})
	}

	// Inject provider API keys if enabled
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
