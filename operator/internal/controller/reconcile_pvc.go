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

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	opencodev1alpha1 "github.com/timothyclin/k8s-opencode/operator/api/v1alpha1"
)

const (
	defaultWorkspacePVCSize = "10Gi"
	defaultDataPVCSize      = "5Gi"
)

func (r *OpenCodeWorkspaceReconciler) reconcilePVCs(ctx context.Context, workspace *opencodev1alpha1.OpenCodeWorkspace, namespaceName string) error {
	workspaceSize := pvcStorageSize(workspace.Spec.Storage.Workspace, defaultWorkspacePVCSize)
	dataSize := pvcStorageSize(workspace.Spec.Storage.Data, defaultDataPVCSize)
	storageClassName := workspace.Spec.Storage.StorageClassName

	workspacePVC := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "workspace-pvc",
			Namespace: namespaceName,
		},
	}

	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, workspacePVC, func() error {
		if err := controllerutil.SetOwnerReference(workspace, workspacePVC, r.Scheme); err != nil {
			return err
		}
		workspacePVC.Spec.AccessModes = []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce}
		workspacePVC.Spec.Resources.Requests = corev1.ResourceList{
			corev1.ResourceStorage: workspaceSize,
		}
		// Only set storageClassName if explicitly configured (it's immutable after creation)
		if storageClassName != "" && workspacePVC.Spec.StorageClassName == nil {
			workspacePVC.Spec.StorageClassName = &storageClassName
		}
		return nil
	})
	if err != nil {
		return err
	}

	dataPVC := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "data-pvc",
			Namespace: namespaceName,
		},
	}

	_, err = controllerutil.CreateOrUpdate(ctx, r.Client, dataPVC, func() error {
		if err := controllerutil.SetOwnerReference(workspace, dataPVC, r.Scheme); err != nil {
			return err
		}
		dataPVC.Spec.AccessModes = []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce}
		dataPVC.Spec.Resources.Requests = corev1.ResourceList{
			corev1.ResourceStorage: dataSize,
		}
		if storageClassName != "" && dataPVC.Spec.StorageClassName == nil {
			dataPVC.Spec.StorageClassName = &storageClassName
		}
		return nil
	})
	if err != nil {
		return err
	}

	return nil
}

func pvcStorageSize(value resource.Quantity, defaultValue string) resource.Quantity {
	if !value.IsZero() {
		return value
	}

	return resource.MustParse(defaultValue)
}
