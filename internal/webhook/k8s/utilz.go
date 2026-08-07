package k8s

import (
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
)

func fixRegistry(registry string, overrides map[string]string) string {
	override, found := overrides[registry]
	if !found {
		return registry
	}
	return override
}

func AlterPodImageRegistry(image string, overrides map[string]string) string {
	data := strings.Split(image, "/")

	isRegistryProvided := len(data) > 1 &&
		(strings.Contains(data[0], ".") || strings.Contains(data[0], ":"))

	if isRegistryProvided {
		registry := fixRegistry(data[0], overrides)
		return fmt.Sprintf("%s/%s", registry, strings.Join(data[1:], "/"))
	}

	return image
}

// Contains - returns true if l contains all the keys with values from r
// returns false otherwise
func Contains(l map[string]string, r map[string]string) bool {
	for k, v := range r {
		val, found := l[k]
		if !found || val != v {
			return false
		}
	}
	return true
}

type ClusterTrustBundle struct {
	Name            string `json:"name" validate:"required"`
	CertWritePath   string `json:"certWritePath" validate:"required"`
	VolumeMountPath string `json:"volumeMountPath" validate:"required"`
	VolumeName      string `json:"volumeName" validate:"required"`
	ResyncInterval  string `json:"resyncInterval,omitempty"`
}

func (r ClusterTrustBundle) ClusterTrustedBundle() corev1.Volume {
	return corev1.Volume{
		Name: r.VolumeName,
		VolumeSource: corev1.VolumeSource{
			Projected: &corev1.ProjectedVolumeSource{
				Sources: []corev1.VolumeProjection{
					{
						ClusterTrustBundle: &corev1.ClusterTrustBundleProjection{
							Name: &r.Name,
							Path: r.CertWritePath,
						},
					},
				},
			},
		},
	}

}

func (r ClusterTrustBundle) VolumeMount() corev1.VolumeMount {
	return corev1.VolumeMount{
		Name:      r.VolumeName,
		ReadOnly:  true,
		MountPath: r.VolumeMountPath,
	}
}

func (r ClusterTrustBundle) KeysAndValues() []any {
	return []any{
		"name", r.Name,
		"volumeName", r.VolumeName,
		"certWritePath", r.CertWritePath,
		"volumeMountPath", r.VolumeMountPath,
	}
}
