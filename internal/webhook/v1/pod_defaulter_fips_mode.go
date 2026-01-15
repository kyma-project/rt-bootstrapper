package v1

import (
	"log/slog"
	"slices"

	apiv1 "github.com/kyma-project/rt-bootstrapper/pkg/api/v1"
	corev1 "k8s.io/api/core/v1"
)

var (
	annotationSetFipsMode = map[string]string{
		apiv1.AnnotationSetFipsMode: "true"}

	envVarKymaFipsModeEnabled = corev1.EnvVar{
		Name:  apiv1.EnvKymaFipsModeEnabled,
		Value: "true"}
)

func BuildDefaulterFipsMode(nsf apiv1.NamespaceFeatures) PodDefaulter {
	handleContainers := func(cs []corev1.Container) bool {
		var modified bool
		for i, c := range cs {
			index := slices.IndexFunc(c.Env, func(v corev1.EnvVar) bool {
				return v.Name == apiv1.EnvKymaFipsModeEnabled
			})
			// env variable not found
			if index == -1 {
				cs[i].Env = append(c.Env, envVarKymaFipsModeEnabled)
				modified = true
				slog.Debug("env variable added",
					"name", apiv1.EnvKymaFipsModeEnabled)
				continue
			}
			// env variable already exists and has the same value
			if cs[i].Env[index].Value == "true" {
				slog.Debug("env variable already exists",
					"name", apiv1.EnvKymaFipsModeEnabled,
				)
				continue
			}
			// env variable already exists but has different value
			slog.Debug("replacing env variable",
				"name", apiv1.EnvKymaFipsModeEnabled,
				"prev", c.Env[i].Value,
			)
			cs[i].Env[index] = envVarKymaFipsModeEnabled
			modified = true
		}
		return modified
	}

	setFipsMode := func(p *corev1.Pod) bool {
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
		namespaceFeatures: nsf,
	})
}
