package v1

import (
	"log/slog"
	"reflect"
	"slices"

	"github.com/kyma-project/rt-bootstrapper/internal/ctb"
	"github.com/kyma-project/rt-bootstrapper/internal/webhook/k8s"
	apiv1 "github.com/kyma-project/rt-bootstrapper/pkg/api/v1"
	corev1 "k8s.io/api/core/v1"
)

type PodDefaulter = func(p *corev1.Pod, nsAnnotations map[string]string, cfg *apiv1.Config) (bool, error)

var (
	annotationsAlterImgRegistry = map[string]string{
		apiv1.AnnotationAlterImgRegistry: "true",
	}
	annotationsSetPullSecret = map[string]string{
		apiv1.AnnotationSetPullSecret: "true",
	}
	annotationAddClusterTrustBundle = map[string]string{
		apiv1.AnnotationAddClusterTrustBundle: "true",
	}
)

type updateOpts struct {
	activeAnnotations map[string]string
}

func defaultPod(update func(*corev1.Pod, *apiv1.Config) bool, opts updateOpts) PodDefaulter {
	return func(p *corev1.Pod, nsAnnotations map[string]string, cfg *apiv1.Config) (bool, error) {
		// prepare logger
		kvs := keysAndValues(p)

		defaultFeatures := cfg.NamespaceDefaultFeatures(p.Namespace)
		expandedNamespaceAnnotations := cfg.ExpandAnnotationAll(nsAnnotations)
		expandedPodAnnotations := cfg.ExpandAnnotationAll(p.Annotations)

		logger := slog.Default().
			WithGroup("args").
			With(kvs...).
			With("ns-annotations", nsAnnotations).
			With("active-features", opts.activeAnnotations).
			With("default-features", defaultFeatures)

		for _, annotations := range []map[string]string{defaultFeatures, expandedNamespaceAnnotations, expandedPodAnnotations} {
			if k8s.Contains(annotations, opts.activeAnnotations) {
				logger.Debug("pod defaulting opt in")
				return update(p, cfg), nil
			}
		}

		logger.Debug("opt out", "ns-annotations", nsAnnotations)
		return false, nil
	}
}

func alterImgRegistry(containers []corev1.Container, overrides map[string]string) bool {
	var modified bool
	for i := range containers {
		alteredImage := k8s.AlterPodImageRegistry(
			containers[i].Image,
			overrides)

		if alteredImage == containers[i].Image {
			continue
		}

		containers[i].Image = alteredImage
		modified = true

		slog.With("overrides", overrides,
			"image-name", containers[i].Image,
			"container-name", containers[i].Name).Debug("image altered")
	}
	return modified
}

func BuildPodDefaulterAlterImgRegistry() PodDefaulter {
	alterPodImageRegistry := func(p *corev1.Pod, c *apiv1.Config) bool {
		var modified bool
		for _, containers := range [][]corev1.Container{
			p.Spec.InitContainers,
			p.Spec.Containers,
		} {
			if !alterImgRegistry(containers, c.Overrides) {
				continue
			}
			modified = true
		}
		return modified
	}

	return defaultPod(alterPodImageRegistry, updateOpts{
		activeAnnotations: annotationsAlterImgRegistry,
	})
}

func BuildPodDefaulterAddImagePullSecrets(secretName string) PodDefaulter {
	addImgPullSecret := func(p *corev1.Pod, c *apiv1.Config) bool {
		imgPullSecret := corev1.LocalObjectReference{Name: secretName}
		if slices.Contains(p.Spec.ImagePullSecrets, imgPullSecret) {
			slog.Debug("image pull secret already found")
			return false
		}

		slog.Debug("adding new image pull secret")
		p.Spec.ImagePullSecrets = append(p.Spec.ImagePullSecrets, imgPullSecret)
		return true
	}

	return defaultPod(addImgPullSecret, updateOpts{
		activeAnnotations: annotationsSetPullSecret,
	})
}

func BuildDefaulterAddClusterTrustBundle(hashHolder *ctb.HashHolder) PodDefaulter {
	// slog.Debug("building volume", mapping.KeysAndValues()...)

	handleVolumeMount := func(cs []corev1.Container, m *k8s.ClusterTrustBundle) bool {
		// stores information if any container was modified
		var result bool
		vol := m.ClusterTrustedBundle()

		for i, c := range cs {
			index := slices.IndexFunc(c.VolumeMounts, func(vm corev1.VolumeMount) bool {
				return vm.Name == m.VolumeName
			})

			if index == -1 {
				vm := m.VolumeMount()
				cs[i].VolumeMounts = append(c.VolumeMounts, vm)
				result = true
				slog.Debug("volume mount added")
				continue
			}

			if reflect.DeepEqual(c.VolumeMounts[index], vol) {
				slog.Debug("volume already mounted, nothing to do")
				continue
			}

			vm := m.VolumeMount()
			cs[i].VolumeMounts[index] = vm
			slog.Debug("volume mount replaced")
			result = true
		}

		return result
	}

	handleContainers := func(modified bool, p *corev1.Pod, m *k8s.ClusterTrustBundle) bool {
		if handleVolumeMount(p.Spec.Containers, m) {
			modified = true
		}

		return modified
	}

	handleClusterTrustBundle := func(p *corev1.Pod, c *apiv1.Config) bool {
		if c.ClusterTrustBundleMapping == nil {
			return false
		}

		index := slices.IndexFunc(p.Spec.Volumes, func(v corev1.Volume) bool {
			return v.Name == c.ClusterTrustBundleMapping.VolumeName
		})

		vol := c.ClusterTrustBundleMapping.ClusterTrustedBundle()

		if index == -1 {
			// volume does not exist, add it
			p.Spec.Volumes = append(p.Spec.Volumes, vol)
			slog.Debug("volume added")
			return handleContainers(true, p, c.ClusterTrustBundleMapping)
		}

		if reflect.DeepEqual(p.Spec.Volumes[index], vol) {
			slog.Debug("volume already added, nothing to do")
			return handleContainers(false, p, c.ClusterTrustBundleMapping)
		}

		p.Spec.Volumes[index] = vol
		slog.Debug("volume replaced")

		return handleContainers(true, p, c.ClusterTrustBundleMapping)
	}

	return func(p *corev1.Pod, nsAnnotations map[string]string, cfg *apiv1.Config) (bool, error) {
		kvs := keysAndValues(p)

		// Explicit opt-out: pod annotation "false" overrides everything, including namespace defaults.
		if apiv1.CTBExplicitOptOut(p.Annotations) {
			slog.Default().WithGroup("args").With(kvs...).Debug("CTB explicit opt out")
			return false, nil
		}

		defaultFeatures := cfg.NamespaceDefaultFeatures(p.Namespace)
		expandedNamespaceAnnotations := cfg.ExpandAnnotationAll(nsAnnotations)
		expandedPodAnnotations := cfg.ExpandAnnotationAll(p.Annotations)

		for _, annotations := range []map[string]string{defaultFeatures, expandedNamespaceAnnotations, expandedPodAnnotations} {
			if apiv1.CTBMountEnabled(annotations) || k8s.Contains(annotations, annotationAddClusterTrustBundle) {
				slog.Default().WithGroup("args").With(kvs...).Debug("CTB pod defaulting opt in")
				modified := handleClusterTrustBundle(p, cfg)
				// Stamp hash annotation on every CTB-opted-in pod when hash is set
				if hash := hashHolder.Get(); hash != "" {
					if p.Annotations == nil {
						p.Annotations = map[string]string{}
					}
					p.Annotations[apiv1.AnnotationCTBHash] = hash
					modified = true
				}
				return modified, nil
			}
		}

		slog.Default().WithGroup("args").With(kvs...).Debug("CTB opt out")
		return false, nil
	}
}
