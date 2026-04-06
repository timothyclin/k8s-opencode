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
	"strings"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"

	opencodev1alpha1 "github.com/timothyclin/k8s-opencode/operator/api/v1alpha1"
)

const (
	emailMapName = "email-map"
	emailMapKey  = "email-map.json"
)

func (r *OpenCodeWorkspaceReconciler) reconcileEmailMap(ctx context.Context, workspace *opencodev1alpha1.OpenCodeWorkspace) error {
	email := strings.TrimSpace(workspace.Spec.Email)
	if email == "" {
		return fmt.Errorf("workspace email is empty")
	}

	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		key := types.NamespacedName{Name: emailMapName, Namespace: r.SystemNamespace}
		configMap := &corev1.ConfigMap{}
		if err := r.Get(ctx, key, configMap); err != nil {
			if !apierrors.IsNotFound(err) {
				return fmt.Errorf("get email-map ConfigMap: %w", err)
			}

			mapping := map[string]string{email: workspace.Name}
			payload, err := json.Marshal(mapping)
			if err != nil {
				return fmt.Errorf("marshal email-map payload: %w", err)
			}

			configMap = &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      emailMapName,
					Namespace: r.SystemNamespace,
				},
				Data: map[string]string{emailMapKey: string(payload)},
			}

			if err := r.Create(ctx, configMap); err != nil {
				if !apierrors.IsAlreadyExists(err) {
					return fmt.Errorf("create email-map ConfigMap: %w", err)
				}
			} else {
				return nil
			}

			if err := r.Get(ctx, key, configMap); err != nil {
				return fmt.Errorf("get email-map ConfigMap after create: %w", err)
			}
		}

		mapping, err := parseEmailMap(configMap.Data[emailMapKey])
		if err != nil {
			return err
		}
		mapping[email] = workspace.Name
		payload, err := json.Marshal(mapping)
		if err != nil {
			return fmt.Errorf("marshal email-map payload: %w", err)
		}

		if configMap.Data == nil {
			configMap.Data = map[string]string{}
		}
		if configMap.Data[emailMapKey] == string(payload) {
			return nil
		}
		configMap.Data[emailMapKey] = string(payload)

		if err := r.Update(ctx, configMap); err != nil {
			return fmt.Errorf("update email-map ConfigMap: %w", err)
		}

		return nil
	})
}

func (r *OpenCodeWorkspaceReconciler) removeFromEmailMap(ctx context.Context, workspace *opencodev1alpha1.OpenCodeWorkspace) error {
	email := strings.TrimSpace(workspace.Spec.Email)
	if email == "" {
		return nil
	}

	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		key := types.NamespacedName{Name: emailMapName, Namespace: r.SystemNamespace}
		configMap := &corev1.ConfigMap{}
		if err := r.Get(ctx, key, configMap); err != nil {
			if apierrors.IsNotFound(err) {
				return nil
			}
			return fmt.Errorf("get email-map ConfigMap: %w", err)
		}

		mapping, err := parseEmailMap(configMap.Data[emailMapKey])
		if err != nil {
			return err
		}
		if _, found := mapping[email]; !found {
			return nil
		}
		delete(mapping, email)

		payload, err := json.Marshal(mapping)
		if err != nil {
			return fmt.Errorf("marshal email-map payload: %w", err)
		}
		if configMap.Data == nil {
			configMap.Data = map[string]string{}
		}
		configMap.Data[emailMapKey] = string(payload)

		if err := r.Update(ctx, configMap); err != nil {
			return fmt.Errorf("update email-map ConfigMap: %w", err)
		}

		return nil
	})
}

func parseEmailMap(raw string) (map[string]string, error) {
	if strings.TrimSpace(raw) == "" {
		return map[string]string{}, nil
	}

	mapping := map[string]string{}
	if err := json.Unmarshal([]byte(raw), &mapping); err != nil {
		return nil, fmt.Errorf("parse email-map.json: %w", err)
	}
	if mapping == nil {
		return map[string]string{}, nil
	}
	return mapping, nil
}
