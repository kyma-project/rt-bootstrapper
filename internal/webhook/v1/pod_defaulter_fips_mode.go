package v1

import (
	"log/slog"
	"slices"

	apiv1 "github.com/kyma-project/rt-bootstrapper/pkg/api/v1"
	corev1 "k8s.io/api/core/v1"
)

var (
	annotationSetFipsMode = map[string]string{
		apiv1.AnnotationSetFipsMode: "true",
	}

	// fipsModeEnvVarNames contains all environment variable names that should be set
	// when FIPS mode is enabled. Both are set for backward compatibility:
	// - KYMA_FIPS_MODE_ENABLED: the canonical Kyma environment variable
	// - FIPS_MODE_ENABLED: legacy variable used by some modules
	fipsModeEnvVarNames = []string{
		apiv1.EnvKymaFipsModeEnabled,
		apiv1.EnvFipsModeEnabled,
	}
)

func BuildDefaulterFipsMode() PodDefaulter {
	handleContainers := func(cs []corev1.Container) bool {
		var modified bool
		for i, c := range cs {
			for _, envName := range fipsModeEnvVarNames {
				if setFipsModeEnvVar(&cs[i], c.Env, envName) {
					modified = true
				}
			}
		}
		return modified
	}

	setFipsMode := func(p *corev1.Pod, _ *apiv1.Config) bool {
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

	return defaultPod(setFipsMode, updateOpts{
		activeAnnotations: annotationSetFipsMode,
	})
}

// setFipsModeEnvVar ensures the given environment variable is set to "true" in the container.
// Returns true if the container was modified.
func setFipsModeEnvVar(container *corev1.Container, currentEnv []corev1.EnvVar, envName string) bool {
	index := slices.IndexFunc(currentEnv, func(v corev1.EnvVar) bool {
		return v.Name == envName
	})

	envVar := corev1.EnvVar{
		Name:  envName,
		Value: "true",
	}

	// env variable not found - add it
	if index == -1 {
		container.Env = append(container.Env, envVar)
		slog.Debug("env variable added", "name", envName)
		return true
	}

	// env variable already exists and has the same value
	if container.Env[index].Value == "true" {
		slog.Debug("env variable already exists", "name", envName)
		return false
	}

	// env variable already exists but has different value - replace it
	slog.Debug("replacing env variable",
		"name", envName,
		"prev", container.Env[index].Value,
	)
	container.Env[index] = envVar
	return true
}
