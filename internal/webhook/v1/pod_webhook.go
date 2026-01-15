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
	"log/slog"

	apiv1 "github.com/kyma-project/rt-bootstrapper/pkg/api/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// SetupPodWebhookWithManager registers the webhook for Pod in the manager.
func SetupPodWebhookWithManager(mgr ctrl.Manager, cfg *apiv1.Config) error {

	slog.Info("setting up webhook", "cfg", cfg)

	nsf := apiv1.NamespaceFeatures{}
	if cfg.NamespaceFeatures != nil {
		nsf = *cfg.NamespaceFeatures
		slog.Info("default features configuration found", "default-features", nsf.Features)
	}

	d1 := BuildPodDefaulterAddImagePullSecrets(cfg.ImagePullSecretName, nsf)
	d2 := BuildPodDefaulterAlterImgRegistry(cfg.Overrides, nsf)
	d3 := BuildDefaulterFipsMode(nsf)

	getNamespace := func(ctx context.Context, name string) (map[string]string, error) {
		var ns corev1.Namespace
		if err := mgr.GetClient().Get(ctx, client.ObjectKey{
			Name: name,
		}, &ns); err != nil {
			return nil, err
		}

		result := ns.Annotations

		slog.Default().WithGroup("get-namespace").Debug("namespace fetched",
			"name", name,
			"annotations", result)

		return result, nil
	}

	defaulter := podCustomDefaulter{
		defaulters: []func(*corev1.Pod, map[string]string) (bool, error){
			d1,
			d2,
			d3,
		},
		GetNsAnnotations: getNamespace,
	}

	// conditional defaulters

	if cfg.ClusterTrustBundleMapping != nil {
		d4 := BuildDefaulterAddClusterTrustBundle(*cfg.ClusterTrustBundleMapping, nsf)
		defaulter.defaulters = append(defaulter.defaulters, d4)
	}

	return ctrl.NewWebhookManagedBy(mgr).For(&corev1.Pod{}).
		WithDefaulter(&defaulter).
		Complete()
}

// +kubebuilder:webhook:path=/mutate--v1-pod,mutating=true,failurePolicy=fail,sideEffects=None,groups="",resources=pods,verbs=create,versions=v1,name=mpod-v1.kb.io,admissionReviewVersions=v1

// +kubebuilder:rbac:groups="",resources=namespaces,verbs=get;list;watch
// +kubebuilder:rbac:groups="admissionregistration.k8s.io",resources=mutatingwebhookconfigurations,verbs=get;patch

// podCustomDefaulter struct is responsible for setting default values on the custom resource of the
// Kind Pod when those are created or updated.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as it is used only for temporary operations and does not need to be deeply copied.
type podCustomDefaulter struct {
	defaulters []func(*corev1.Pod, map[string]string) (bool, error)
	GetNsAnnotations
}

var _ webhook.CustomDefaulter = &podCustomDefaulter{}

type GetNsAnnotations = func(context.Context, string) (map[string]string, error)

// Default implements webhook.CustomDefaulter so a webhook will be registered for the Kind Pod.
func (d *podCustomDefaulter) Default(ctx context.Context, obj runtime.Object) (err error) {
	defer func() {
		r := recover()
		if r == nil {
			// no panic
			return
		}

		switch x := r.(type) {
		case string:
			err = fmt.Errorf("%s", x)
		case error:
			err = x
		default:
			err = fmt.Errorf("unknown defaulting function panic: %s", r)
		}
	}()

	pod, ok := obj.(*corev1.Pod)
	if !ok {
		return fmt.Errorf("expected an Pod object but got %T", obj)
	}

	nsAnnotations, err := d.GetNsAnnotations(ctx, pod.Namespace)
	if err != nil {
		slog.Error("unable to get namespace", "error", err)
		return err
	}

	var podDefaulted bool
	for i, defaulter := range d.defaulters {
		kvals := keysAndValues(pod)
		slog.Default().WithGroup("pod").With(kvals...).
			WithGroup("ns").With("annotations", nsAnnotations).
			WithGroup("for").Debug("invoking defaulter",
			"i", fmt.Sprintf("%d", i))

		podModified, err := defaulter(pod, nsAnnotations)
		if err != nil {
			return err
		}

		if !podModified {
			continue
		}

		podDefaulted = true
	}

	if !podDefaulted {
		return nil
	}

	if pod.Annotations == nil {
		pod.Annotations = map[string]string{}
	}
	pod.Annotations[apiv1.AnnotationDefaulted] = "true"

	return nil
}

func keysAndValues(pod *corev1.Pod) []any {
	return []any{
		"name", pod.GetGenerateName(),
		"ns", pod.GetNamespace(),
		"pod-annotations", pod.GetAnnotations(),
	}
}
