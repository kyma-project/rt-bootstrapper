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

package controller

import (
	"context"
	"errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	apiv1 "github.com/kyma-project/rt-bootstrapper/pkg/api/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"k8s.io/client-go/kubernetes/scheme"
)

var _ = Describe("Secret Controller", func() {
	Context("When reconciling a resource", func() {

		It("should successfully reconcile the resource", func() {

			// TODO(user): Add more specific assertions depending on your controller's reconciliation logic.
			// Example: If you expect a certain status condition after reconciliation, verify it here.
		})
	})

	Context("Reconcile feature gate", func() {
		req := ctrl.Request{NamespacedName: types.NamespacedName{Name: "test-ns"}}

		It("should skip reconcile when AnnotationSetPullSecret is not in availableFeatures", func() {
			r := &SecretReconciler{
				Client: fake.NewClientBuilder().WithScheme(scheme.Scheme).Build(),
				NamespacedName: types.NamespacedName{Name: "registry-credentials", Namespace: "kyma-system"},
				GetConfig: func(_ context.Context) (*apiv1.Config, error) {
					return &apiv1.Config{
						Overrides:         map[string]string{},
						AvailableFeatures: []string{apiv1.AnnotationAlterImgRegistry},
					}, nil
				},
			}

			result, err := r.Reconcile(ctx, req)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(result).Should(Equal(ctrl.Result{}))
		})

		It("should return error when GetConfig fails", func() {
			configErr := errors.New("configmap not found")
			r := &SecretReconciler{
				Client: fake.NewClientBuilder().WithScheme(scheme.Scheme).Build(),
				NamespacedName: types.NamespacedName{Name: "registry-credentials", Namespace: "kyma-system"},
				GetConfig: func(_ context.Context) (*apiv1.Config, error) {
					return nil, configErr
				},
			}

			_, err := r.Reconcile(ctx, req)
			Expect(err).Should(HaveOccurred())
			Expect(err.Error()).Should(ContainSubstring("configmap not found"))
		})
	})
})
