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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
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
			// Cleanup logic after each test, like removing the resource instance.
			resource := &opencodev1alpha1.OpenCodeWorkspace{}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			if err != nil {
				if errors.IsNotFound(err) {
					// Resource was already deleted or never created
					return
				}
				Fail(fmt.Sprintf("Failed to get resource for cleanup: %v", err))
			}

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

			By("Checking the resource status")
			// Re-fetch the resource to get updated status
			err = k8sClient.Get(ctx, typeNamespacedName, opencodeworkspace)
			Expect(err).NotTo(HaveOccurred())

			// Verify that the status has been updated
			Expect(opencodeworkspace.Status.Phase).NotTo(BeEmpty(), "Status phase should be set")
			Expect(opencodeworkspace.Status.Namespace).To(Equal("oc-test-resource"), "Status namespace should be set correctly")
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

		It("should set Command field to start OpenCode server", func() {
			By("Creating the test workspace")
			resource := &opencodev1alpha1.OpenCodeWorkspace{
				ObjectMeta: metav1.ObjectMeta{
					Name: resourceName + "-command-test",
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

			By("Checking that StatefulSet has correct Command")
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
			Expect(container.Command).NotTo(BeEmpty(), "Container should have Command set")
			Expect(container.Command).To(Equal([]string{"opencode", "serve", "--hostname", "0.0.0.0", "--port", "4096"}))

			By("Cleaning up the test workspace")
			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
		})

		It("should set securityContext to run as non-root user", func() {
			By("Creating the test workspace")
			resource := &opencodev1alpha1.OpenCodeWorkspace{
				ObjectMeta: metav1.ObjectMeta{
					Name: resourceName + "-security-test",
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

			By("Checking that StatefulSet has correct securityContext")
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

			podSpec := statefulSet.Spec.Template.Spec
			Expect(podSpec.SecurityContext).NotTo(BeNil(), "Pod should have securityContext")

			runAsNonRoot := true
			var runAsUser int64 = 1000
			var runAsGroup int64 = 1000
			var fsGroup int64 = 1000

			Expect(podSpec.SecurityContext.RunAsNonRoot).To(Equal(&runAsNonRoot))
			Expect(podSpec.SecurityContext.RunAsUser).To(Equal(&runAsUser))
			Expect(podSpec.SecurityContext.RunAsGroup).To(Equal(&runAsGroup))
			Expect(podSpec.SecurityContext.FSGroup).To(Equal(&fsGroup))

			By("Cleaning up the test workspace")
			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
		})

		It("should configure init-permissions initContainer for user setup", func() {
			ctx := context.Background()

			By("Creating the custom resource")
			resource := &opencodev1alpha1.OpenCodeWorkspace{
				ObjectMeta: metav1.ObjectMeta{
					Name: resourceName + "-init-test",
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

			By("Checking that StatefulSet has init-permissions initContainer")
			expectedNamespace := "oc-" + resource.Name
			statefulSet := &appsv1.StatefulSet{}
			statefulSetName := types.NamespacedName{
				Name:      "workspace",
				Namespace: expectedNamespace,
			}
			err := k8sClient.Get(ctx, statefulSetName, statefulSet)
			Expect(err).NotTo(HaveOccurred())

			podSpec := statefulSet.Spec.Template.Spec
			Expect(podSpec.InitContainers).To(HaveLen(1), "Pod should have 1 initContainer")

			initContainer := podSpec.InitContainers[0]
			Expect(initContainer.Name).To(Equal("init-permissions"))
			Expect(initContainer.SecurityContext).NotTo(BeNil())
			Expect(initContainer.SecurityContext.RunAsUser).To(Equal(ptr.To(int64(0))))
			Expect(initContainer.SecurityContext.RunAsNonRoot).To(Equal(ptr.To(false)))

			By("Checking that initContainer has required volume mounts")
			mountNames := make(map[string]bool)
			for _, mount := range initContainer.VolumeMounts {
				mountNames[mount.Name] = true
			}
			Expect(mountNames).To(HaveKey("sudoers"))
			Expect(mountNames).To(HaveKey("passwd"))
			Expect(mountNames).To(HaveKey("shadow"))

			By("Checking that main container has passwd/shadow/sudoers mounts")
			mainContainer := podSpec.Containers[0]
			mainMountNames := make(map[string]bool)
			for _, mount := range mainContainer.VolumeMounts {
				mainMountNames[mount.Name] = true
			}
			Expect(mainMountNames).To(HaveKey("sudoers"))
			Expect(mainMountNames).To(HaveKey("passwd"))
			Expect(mainMountNames).To(HaveKey("shadow"))

			By("Checking that pod has emptyDir volumes")
			volumeNames := make(map[string]bool)
			for _, volume := range podSpec.Volumes {
				volumeNames[volume.Name] = true
			}
			Expect(volumeNames).To(HaveKey("sudoers"))
			Expect(volumeNames).To(HaveKey("passwd"))
			Expect(volumeNames).To(HaveKey("shadow"))

			By("Cleaning up the test workspace")
			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
		})
	})
})
