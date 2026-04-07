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
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	opencodev1alpha1 "github.com/timothyclin/k8s-opencode/operator/api/v1alpha1"
)

const (
	finalizerName = "opencode.io/workspace-finalizer"

	phasePending     = "Pending"
	phaseCreating    = "Creating"
	phaseReconciling = "Reconciling"
	phaseRunning     = "Running"
	phaseFailed      = "Failed"
	phaseTerminating = "Terminating"

	conditionTypeReady       = "Ready"
	conditionTypeProgressing = "Progressing"
	conditionTypeDegraded    = "Degraded"
)

type OpenCodeWorkspaceReconciler struct {
	client.Client
	Scheme          *runtime.Scheme
	SystemNamespace string
}

// +kubebuilder:rbac:groups=opencode.opencode.io,resources=opencodeworkspaces,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=opencode.opencode.io,resources=opencodeworkspaces/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=opencode.opencode.io,resources=opencodeworkspaces/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=namespaces,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=persistentvolumeclaims,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=statefulsets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=networking.k8s.io,resources=networkpolicies,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=networking.k8s.io,resources=ingresses,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=networking.k8s.io,resources=ingressclasses,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=serviceaccounts,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=roles,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=rolebindings,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=pods;pods/exec;pods/log,verbs=create;delete;get;list;patch;update;watch

func (r *OpenCodeWorkspaceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	workspace := &opencodev1alpha1.OpenCodeWorkspace{}
	if err := r.Get(ctx, req.NamespacedName, workspace); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if !workspace.DeletionTimestamp.IsZero() {
		return r.handleDeletion(ctx, workspace)
	}

	if !controllerutil.ContainsFinalizer(workspace, finalizerName) {
		controllerutil.AddFinalizer(workspace, finalizerName)
		if err := r.Update(ctx, workspace); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	if workspace.Status.Phase == "" {
		workspace.Status.Phase = phasePending
		if err := r.Status().Update(ctx, workspace); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	if workspace.Status.Phase == phasePending {
		workspace.Status.Phase = phaseCreating
		r.setCondition(workspace, conditionTypeProgressing, metav1.ConditionTrue, "CreatingResources", "Creating workspace resources")
		if err := r.Status().Update(ctx, workspace); err != nil {
			return ctrl.Result{}, err
		}
	}

	namespaceName, err := r.reconcileNamespace(ctx, workspace)
	if err != nil {
		return r.handleReconcileError(ctx, workspace, "Namespace", err)
	}

	if workspace.Status.Namespace != namespaceName {
		workspace.Status.Namespace = namespaceName
		if err := r.Status().Update(ctx, workspace); err != nil {
			return ctrl.Result{}, err
		}
	}

	log.Info("Reconciling workspace", "name", workspace.Name, "namespace", namespaceName)

	if err := r.reconcileNetworkPolicy(ctx, workspace, namespaceName); err != nil {
		return r.handleReconcileError(ctx, workspace, "NetworkPolicy", err)
	}

	apiKeys, err := r.reconcileSecrets(ctx, workspace, namespaceName)
	if err != nil {
		return r.handleReconcileError(ctx, workspace, "Secrets", err)
	}

	apiKeysMap := map[string]string{
		"anthropic":  apiKeys.Anthropic,
		"openai":     apiKeys.OpenAI,
		"openrouter": apiKeys.OpenRouter,
	}

	if err := r.reconcileConfigMap(ctx, workspace, namespaceName, apiKeysMap); err != nil {
		return r.handleReconcileError(ctx, workspace, "ConfigMap", err)
	}

	if err := r.reconcilePVCs(ctx, workspace, namespaceName); err != nil {
		return r.handleReconcileError(ctx, workspace, "PVCs", err)
	}

	if err := r.reconcileKubedock(ctx, workspace, namespaceName); err != nil {
		return r.handleReconcileError(ctx, workspace, "Kubedock", err)
	}

	if err := r.reconcileStatefulSet(ctx, workspace, namespaceName); err != nil {
		return r.handleReconcileError(ctx, workspace, "StatefulSet", err)
	}

	if err := r.reconcileService(ctx, workspace, namespaceName); err != nil {
		return r.handleReconcileError(ctx, workspace, "Service", err)
	}

	if err := r.reconcileTailscaleIngress(ctx, workspace, namespaceName); err != nil {
		return r.handleReconcileError(ctx, workspace, "TailscaleIngress", err)
	}

	if err := r.reconcileEmailMap(ctx, workspace); err != nil {
		return r.handleReconcileError(ctx, workspace, "EmailMap", err)
	}

	workspace.Status.Phase = phaseRunning
	workspace.Status.Message = "Workspace is running"
	r.setCondition(workspace, conditionTypeReady, metav1.ConditionTrue, "WorkspaceReady", "All resources reconciled successfully")
	r.setCondition(workspace, conditionTypeProgressing, metav1.ConditionFalse, "ReconcileComplete", "Reconciliation complete")
	r.setCondition(workspace, conditionTypeDegraded, metav1.ConditionFalse, "Healthy", "No degradation detected")

	if err := r.Status().Update(ctx, workspace); err != nil {
		return ctrl.Result{}, err
	}

	log.Info("Reconciliation complete", "name", workspace.Name)
	return ctrl.Result{}, nil
}

func (r *OpenCodeWorkspaceReconciler) handleDeletion(ctx context.Context, workspace *opencodev1alpha1.OpenCodeWorkspace) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	if !controllerutil.ContainsFinalizer(workspace, finalizerName) {
		return ctrl.Result{}, nil
	}

	workspace.Status.Phase = phaseTerminating
	workspace.Status.Message = "Workspace is being deleted"
	if err := r.Status().Update(ctx, workspace); err != nil {
		return ctrl.Result{}, err
	}

	log.Info("Cleaning up workspace resources", "name", workspace.Name)

	if err := r.removeFromEmailMap(ctx, workspace); err != nil {
		log.Error(err, "Failed to remove from email map", "name", workspace.Name)
	}

	nsPrefix := workspace.Spec.NamespacePrefix
	if nsPrefix == "" {
		nsPrefix = "oc"
	}
	namespaceName := nsPrefix + workspace.Name
	ns := &corev1.Namespace{}
	if err := r.Get(ctx, client.ObjectKey{Name: namespaceName}, ns); err == nil {
		if err := r.Delete(ctx, ns); err != nil && !apierrors.IsNotFound(err) {
			return ctrl.Result{}, fmt.Errorf("failed to delete namespace %q: %w", namespaceName, err)
		}
		log.Info("Deleted namespace", "name", namespaceName)
	}

	controllerutil.RemoveFinalizer(workspace, finalizerName)
	if err := r.Update(ctx, workspace); err != nil {
		return ctrl.Result{}, err
	}

	log.Info("Finalizer removed, workspace cleanup complete", "name", workspace.Name)
	return ctrl.Result{}, nil
}

func (r *OpenCodeWorkspaceReconciler) handleReconcileError(ctx context.Context, workspace *opencodev1alpha1.OpenCodeWorkspace, resource string, err error) (ctrl.Result, error) {
	log := logf.FromContext(ctx)
	log.Error(err, "Failed to reconcile resource", "resource", resource)

	workspace.Status.Phase = phaseFailed
	workspace.Status.Message = fmt.Sprintf("Failed to reconcile %s: %v", resource, err)
	r.setCondition(workspace, conditionTypeDegraded, metav1.ConditionTrue, "ReconcileFailed", workspace.Status.Message)
	r.setCondition(workspace, conditionTypeReady, metav1.ConditionFalse, "ReconcileFailed", workspace.Status.Message)

	if statusErr := r.Status().Update(ctx, workspace); statusErr != nil {
		log.Error(statusErr, "Failed to update status after error")
	}

	return ctrl.Result{}, err
}

func (r *OpenCodeWorkspaceReconciler) setCondition(workspace *opencodev1alpha1.OpenCodeWorkspace, conditionType string, status metav1.ConditionStatus, reason, message string) {
	meta.SetStatusCondition(&workspace.Status.Conditions, metav1.Condition{
		Type:               conditionType,
		Status:             status,
		Reason:             reason,
		Message:            message,
		ObservedGeneration: workspace.Generation,
	})
}

func (r *OpenCodeWorkspaceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&opencodev1alpha1.OpenCodeWorkspace{}).
		Named("opencodeworkspace").
		Complete(r)
}
