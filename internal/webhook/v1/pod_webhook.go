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
	"slices"
	"strings"

	apiv1 "github.com/kyma-project/rt-bootstrapper/pkg/api/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

type SetupPodWebhookWithManagerOpts struct {
	AvailableFeatures        []string
	NamespaceDefaultFeatures func(string) map[string]string
	ImagePullSecretName      string
	Landscape                string
	GetConfig
}

// SetupPodWebhookWithManager registers the webhook for Pod in the manager.
func SetupPodWebhookWithManager(mgr ctrl.Manager, opts SetupPodWebhookWithManagerOpts) error {
	type defaulterEntry struct {
		annotation string
		defaulter  PodDefaulter
	}

	defaulterEntries := []defaulterEntry{
		{annotation: apiv1.AnnotationAlterImgRegistry, defaulter: BuildPodDefaulterAlterImgRegistry()},
		{annotation: apiv1.AnnotationSetPullSecret, defaulter: BuildPodDefaulterAddImagePullSecrets(opts.ImagePullSecretName)},
		{annotation: apiv1.AnnotationAddClusterTrustBundle, defaulter: BuildDefaulterAddClusterTrustBundle()},
		{annotation: apiv1.AnnotationSetFipsMode, defaulter: BuildDefaulterFipsMode()},
		{annotation: apiv1.AnnotationSetLandscape, defaulter: BuildDefaulterSetLandscape(opts.Landscape)},
	}

	// prepare available defaulters
	defaulters := slices.Collect(func(yield func(PodDefaulter) bool) {
		for _, d := range defaulterEntries {
			if slices.Contains(opts.AvailableFeatures, d.annotation) {
				if !yield(d.defaulter) {
					return
				}
			}
		}
	})

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
		defaulters:               defaulters,
		GetConfig:                opts.GetConfig,
		GetNsAnnotations:         getNamespace,
		availableFeatures:        opts.AvailableFeatures,
		namespaceDefaultFeatures: opts.NamespaceDefaultFeatures,
	}

	return ctrl.NewWebhookManagedBy(mgr).For(&corev1.Pod{}).
		WithDefaulter(&defaulter).
		Complete()
}

type GetConfig = func(context.Context) (*apiv1.Config, error)

// +kubebuilder:webhook:path=/mutate--v1-pod,mutating=true,failurePolicy=fail,sideEffects=None,groups="",resources=pods,verbs=create,versions=v1,name=mpod-v1.kb.io,admissionReviewVersions=v1
// +kubebuilder:rbac:groups="",resources=namespaces;configmaps,verbs=get;list;watch
// +kubebuilder:rbac:groups="admissionregistration.k8s.io",resources=mutatingwebhookconfigurations,verbs=get;patch

// podCustomDefaulter struct is responsible for setting default values on the custom resource of the
// Kind Pod when those are created or updated.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as it is used only for temporary operations and does not need to be deeply copied.
type podCustomDefaulter struct {
	namespaceDefaultFeatures func(string) map[string]string
	availableFeatures        []string
	defaulters               []PodDefaulter
	GetNsAnnotations
	GetConfig
}

var _ webhook.CustomDefaulter = &podCustomDefaulter{}

type GetNsAnnotations = func(context.Context, string) (map[string]string, error)

type annotationSource struct {
	level       string
	annotations map[string]string
}

type annotationSources []annotationSource

func (s annotationSources) auditFeatures(availableFeatures []string) {
	for _, src := range s {
		for key := range src.annotations {
			isAvailableFeature := slices.Contains(availableFeatures, key)
			isRtBootstrapperFeature := strings.HasPrefix(key, "rt-bootstrapper.kyma-project.io/")
			if !isAvailableFeature && !isRtBootstrapperFeature {
				slog.Warn("found inactive feature", "key", key)
			}
		}
	}
}

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

	namespaceDefaultFeatures := d.namespaceDefaultFeatures(pod.Namespace)

	// audit for inactive features
	annotationSources := annotationSources{
		{level: "namespace", annotations: namespaceDefaultFeatures},
		{level: "ns", annotations: nsAnnotations},
		{level: "pod", annotations: pod.Annotations},
	}
	annotationSources.auditFeatures(d.availableFeatures)

	var podDefaulted bool
	for i, defaulter := range d.defaulters {
		kvals := keysAndValues(pod)
		slog.Default().WithGroup("pod").With(kvals...).
			WithGroup("ns").With("annotations", nsAnnotations).
			WithGroup("for").Debug("invoking defaulter",
			"i", fmt.Sprintf("%d", i))

		cfg, err := d.GetConfig(ctx)
		if err != nil {
			return err
		}

		podModified, err := defaulter(pod, nsAnnotations, cfg)
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
	pod.Annotations[apiv1.AnnotationModified] = "true"

	return nil
}

func keysAndValues(pod *corev1.Pod) []any {
	return []any{
		"name", pod.GetGenerateName(),
		"ns", pod.GetNamespace(),
		"pod-annotations", pod.GetAnnotations(),
	}
}
