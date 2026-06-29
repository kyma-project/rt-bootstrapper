/*
Copyright 2025.

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

package v1

import (
	"context"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	apiv1 "github.com/kyma-project/rt-bootstrapper/pkg/api/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	testRegistryName = "test-registry"
	testPullSecret   = "test-pull-secret"
)

func getTestPod(annotations map[string]string) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: annotations,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Image: "test.com/me/plz:now",
				},
				{
					Image: "test.com:2000/this/too:plz",
				},
			},
		},
	}
}

var _ = Describe("Pod Webhook", func() {
	Context("When creating Pod under Defaulting Webhook", func() {
		d1 := BuildPodDefaulterAddImagePullSecrets(testPullSecret)
		d2 := BuildPodDefaulterAlterImgRegistry()

		cfg := &apiv1.Config{
			NamespaceFeatures: &apiv1.NamespaceFeatures{},
			Overrides: map[string]string{
				"test.com":      testRegistryName,
				"test.com:2000": testRegistryName,
			},
		}
		defaulter := podCustomDefaulter{
			availableFeatures: []string{
				apiv1.AnnotationAlterImgRegistry,
				apiv1.AnnotationSetPullSecret,
			},
			defaulters: []PodDefaulter{
				d1, d2,
			},
			GetNsAnnotations: func(_ context.Context, name string) (map[string]string, error) {
				return nil, nil
			},
			GetConfig: func(_ context.Context) (*apiv1.Config, error) {
				return cfg, nil
			},
			namespaceDefaultFeatures: cfg.NamespaceDefaultFeatures,
		}

		It("Should alter image registry", func() {
			By(fmt.Sprintf("adding '%s' annotation", apiv1.AnnotationAlterImgRegistry))
			pod := getTestPod(
				map[string]string{apiv1.AnnotationAlterImgRegistry: "true"})
			Expect(pod.Spec.Containers).ShouldNot(BeEmpty())

			By("calling the Default method to alter registry image")
			err := defaulter.Default(ctx, pod)
			Expect(err).ShouldNot(HaveOccurred())

			By("checking that the image was altered")
			for _, container := range pod.Spec.Containers {
				Expect(container.Image).Should(HavePrefix(testRegistryName))
			}
		})

		It("Should add image pull secret", func() {
			By(fmt.Sprintf("adding '%s' label", apiv1.AnnotationSetPullSecret))
			pod := getTestPod(
				map[string]string{apiv1.AnnotationSetPullSecret: "true"})
			Expect(pod.Spec.Containers).ShouldNot(BeEmpty())

			By("calling the Default method to add pull secret")
			err := defaulter.Default(ctx, pod)
			Expect(err).ShouldNot(HaveOccurred())

			By(fmt.Sprintf("checking that the pod's image pull secrets contain '%s'", testPullSecret))
			Expect(pod.Spec.ImagePullSecrets).Should(ContainElement(
				corev1.LocalObjectReference{Name: testPullSecret},
			))
		})

		It("Should opt in both defaulters from namespaceFeatures all alias without pod annotations", func() {
			ns := "kyma-system"
			nsFeat := apiv1.NamespaceFeatures{
				ns: {apiv1.AnnotationAll},
			}
			cfgAll := &apiv1.Config{
				NamespaceFeatures: &nsFeat,
				AvailableFeatures: []string{
					apiv1.AnnotationAlterImgRegistry,
					apiv1.AnnotationSetPullSecret,
				},
				Overrides: map[string]string{
					"test.com":      testRegistryName,
					"test.com:2000": testRegistryName,
				},
			}
			defaulterAll := podCustomDefaulter{
				availableFeatures: []string{
					apiv1.AnnotationAlterImgRegistry,
					apiv1.AnnotationSetPullSecret,
				},
				defaulters: []PodDefaulter{
					d1, d2,
				},
				GetNsAnnotations: func(_ context.Context, name string) (map[string]string, error) {
					return nil, nil
				},
				GetConfig: func(_ context.Context) (*apiv1.Config, error) {
					return cfgAll, nil
				},
				namespaceDefaultFeatures: cfgAll.NamespaceDefaultFeatures,
			}

			pod := getTestPod(nil)
			pod.Namespace = ns

			err := defaulterAll.Default(ctx, pod)
			Expect(err).ShouldNot(HaveOccurred())
			for _, container := range pod.Spec.Containers {
				Expect(container.Image).Should(HavePrefix(testRegistryName))
			}
			Expect(pod.Spec.ImagePullSecrets).Should(ContainElement(
				corev1.LocalObjectReference{Name: testPullSecret},
			))
		})

		It("Should opt in both defaulters from pod annotation rt-cfg all alias", func() {
			cfgPodAll := &apiv1.Config{
				AvailableFeatures: []string{
					apiv1.AnnotationAlterImgRegistry,
					apiv1.AnnotationSetPullSecret,
				},
				Overrides: map[string]string{
					"test.com":      testRegistryName,
					"test.com:2000": testRegistryName,
				},
			}
			defaulterPodAll := podCustomDefaulter{
				availableFeatures: []string{
					apiv1.AnnotationAlterImgRegistry,
					apiv1.AnnotationSetPullSecret,
					apiv1.AnnotationAll,
				},
				defaulters: []PodDefaulter{
					d1, d2,
				},
				GetNsAnnotations: func(_ context.Context, name string) (map[string]string, error) {
					return nil, nil
				},
				GetConfig: func(_ context.Context) (*apiv1.Config, error) {
					return cfgPodAll, nil
				},
				namespaceDefaultFeatures: cfgPodAll.NamespaceDefaultFeatures,
			}

			pod := getTestPod(map[string]string{apiv1.AnnotationAll: "true"})

			err := defaulterPodAll.Default(ctx, pod)
			Expect(err).ShouldNot(HaveOccurred())
			for _, container := range pod.Spec.Containers {
				Expect(container.Image).Should(HavePrefix(testRegistryName))
			}
			Expect(pod.Spec.ImagePullSecrets).Should(ContainElement(
				corev1.LocalObjectReference{Name: testPullSecret},
			))
		})

		It("Should inject landscape env var", func() {
			d5 := BuildDefaulterSetLandscape("NS2")
			cfgLandscape := &apiv1.Config{
				NamespaceFeatures: &apiv1.NamespaceFeatures{},
				Overrides: map[string]string{
					"test.com":      testRegistryName,
					"test.com:2000": testRegistryName,
				},
			}
			defaulterLandscape := podCustomDefaulter{
				availableFeatures: []string{
					apiv1.AnnotationSetLandscape,
				},
				defaulters: []PodDefaulter{
					d5,
				},
				GetNsAnnotations: func(_ context.Context, name string) (map[string]string, error) {
					return nil, nil
				},
				GetConfig: func(_ context.Context) (*apiv1.Config, error) {
					return cfgLandscape, nil
				},
				namespaceDefaultFeatures: cfgLandscape.NamespaceDefaultFeatures,
			}

			By(fmt.Sprintf("adding '%s' annotation", apiv1.AnnotationSetLandscape))
			pod := getTestPod(
				map[string]string{apiv1.AnnotationSetLandscape: "true"})

			By("calling the Default method to set landscape")
			err := defaulterLandscape.Default(ctx, pod)
			Expect(err).ShouldNot(HaveOccurred())

			By("checking that the landscape env var was added")
			for _, container := range pod.Spec.Containers {
				Expect(container.Env).Should(ContainElement(corev1.EnvVar{
					Name:  apiv1.EnvKymaLandscape,
					Value: "NS2",
				}))
			}

			By("checking that the modified annotation was set")
			Expect(pod.Annotations[apiv1.AnnotationModified]).Should(Equal("true"))
		})
	})
})
