package v1

import (
	"log/slog"
	"slices"

	apiv1 "github.com/kyma-project/rt-bootstrapper/pkg/api/v1"
	corev1 "k8s.io/api/core/v1"
)

var (
	annotationSetLandscape = map[string]string{
		apiv1.AnnotationSetLandscape: "true",
	}
)

func BuildDefaulterSetLandscape(landscape string) PodDefaulter {
	handleContainers := func(cs []corev1.Container) bool {
		var modified bool
		for i, c := range cs {
			if setLandscapeEnvVar(&cs[i], c.Env, landscape) {
				modified = true
			}
		}
		return modified
	}

	setLandscape := func(p *corev1.Pod, _ *apiv1.Config) bool {
		var modified bool
		for _, cs := range [][]corev1.Container{
			p.Spec.InitContainers,
			p.Spec.Containers,
		} {
			if handleContainers(cs) {
				modified = true
			}
		}
		return modified
	}

	return defaultPod(setLandscape, updateOpts{
		activeAnnotations: annotationSetLandscape,
	})
}

// setLandscapeEnvVar ensures KYMA_LANDSCAPE is set to the given landscape value.
// Returns true if the container was modified.
func setLandscapeEnvVar(container *corev1.Container, currentEnv []corev1.EnvVar, landscape string) bool {
	envName := apiv1.EnvKymaLandscape

	index := slices.IndexFunc(currentEnv, func(v corev1.EnvVar) bool {
		return v.Name == envName
	})

	envVar := corev1.EnvVar{
		Name:  envName,
		Value: landscape,
	}

	if index == -1 {
		container.Env = append(container.Env, envVar)
		slog.Debug("env variable added", "name", envName, "value", landscape)
		return true
	}

	if container.Env[index].Value == landscape {
		slog.Debug("env variable already exists with correct value", "name", envName)
		return false
	}

	slog.Debug("replacing env variable",
		"name", envName,
		"prev", container.Env[index].Value,
		"new", landscape,
	)
	container.Env[index] = envVar
	return true
}
