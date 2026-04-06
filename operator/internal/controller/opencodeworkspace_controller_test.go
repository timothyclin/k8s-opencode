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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	opencodev1alpha1 "github.com/timothyclin/k8s-opencode/operator/api/v1alpha1"
)

var _ = Describe("OpenCodeWorkspace Controller", func() {
	Context("When reconciling a resource", func() {
		const resourceName = "test-resource"

		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name: resourceName,
		}
		opencodeworkspace := &opencodev1alpha1.OpenCodeWorkspace{}

		BeforeEach(func() {
			By("creating the custom resource for the Kind OpenCodeWorkspace")
			err := k8sClient.Get(ctx, typeNamespacedName, opencodeworkspace)
			if err != nil && errors.IsNotFound(err) {
				resource := &opencodev1alpha1.OpenCodeWorkspace{
					ObjectMeta: metav1.ObjectMeta{
						Name: resourceName,
					},
					Spec: opencodev1alpha1.OpenCodeWorkspaceSpec{
						Email: "test@example.com",
					},
				}
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			}
		})

		AfterEach(func() {
			// TODO(user): Cleanup logic after each test, like removing the resource instance.
			resource := &opencodev1alpha1.OpenCodeWorkspace{}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			Expect(err).NotTo(HaveOccurred())

			By("Cleanup the specific resource instance OpenCodeWorkspace")
			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
		})
		It("should successfully reconcile the resource", func() {
			By("Reconciling the created resource")
			controllerReconciler := &OpenCodeWorkspaceReconciler{
				Client:          k8sClient,
				Scheme:          k8sClient.Scheme(),
				SystemNamespace: "default",
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())
			// TODO(user): Add more specific assertions depending on your controller's reconciliation logic.
			// Example: If you expect a certain status condition after reconciliation, verify it here.
		})

		It("should inject OPENCODE_SERVER_PASSWORD env var into StatefulSet", func() {
			By("Creating the workspace resource")
			resource := &opencodev1alpha1.OpenCodeWorkspace{
				ObjectMeta: metav1.ObjectMeta{
					Name: resourceName + "-password-test",
				},
				Spec: opencodev1alpha1.OpenCodeWorkspaceSpec{
					Email: "test@example.com",
				},
			}
			Expect(k8sClient.Create(ctx, resource)).To(Succeed())

			By("Reconciling the created resource")
			controllerReconciler := &OpenCodeWorkspaceReconciler{
				Client:          k8sClient,
				Scheme:          k8sClient.Scheme(),
				SystemNamespace: "default",
			}

			testNamespacedName := types.NamespacedName{
				Name: resource.Name,
			}

			// Reconcile multiple times to progress through all phases
			for range 5 {
				_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
					NamespacedName: testNamespacedName,
				})
				Expect(err).NotTo(HaveOccurred())
			}

			By("Checking that StatefulSet has OPENCODE_SERVER_PASSWORD")
			// Namespace is created with pattern: {prefix}-{workspace-name}
			// Default prefix is "oc"
			expectedNamespace := "oc-" + resource.Name
			statefulSet := &appsv1.StatefulSet{}
			statefulSetName := types.NamespacedName{
				Name:      "workspace",
				Namespace: expectedNamespace,
			}
			err := k8sClient.Get(ctx, statefulSetName, statefulSet)
			Expect(err).NotTo(HaveOccurred())

			container := statefulSet.Spec.Template.Spec.Containers[0]
			var foundServerPassword bool
			for _, env := range container.Env {
				if env.Name == "OPENCODE_SERVER_PASSWORD" {
					foundServerPassword = true
					Expect(env.ValueFrom).NotTo(BeNil(), "OPENCODE_SERVER_PASSWORD should reference a secret")
					Expect(env.ValueFrom.SecretKeyRef).NotTo(BeNil())
					Expect(env.ValueFrom.SecretKeyRef.Name).To(Equal("workspace-secrets"))
					Expect(env.ValueFrom.SecretKeyRef.Key).To(Equal("server-password"))
					break
				}
			}
			Expect(foundServerPassword).To(BeTrue(), "StatefulSet should have OPENCODE_SERVER_PASSWORD env var")

			By("Cleaning up the test workspace")
			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
		})
	})
})
