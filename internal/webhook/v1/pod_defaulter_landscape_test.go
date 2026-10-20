package v1

import (
	"testing"

	apiv1 "github.com/kyma-project/rt-bootstrapper/pkg/api/v1"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestSetLandscapeEnvVar(t *testing.T) {
	t.Run("adds env var when not present", func(t *testing.T) {
		container := &corev1.Container{Name: "test"}
		modified := setLandscapeEnvVar(container, container.Env, "NS2")
		assert.True(t, modified)
		assert.Contains(t, container.Env, corev1.EnvVar{
			Name:  apiv1.EnvKymaLandscape,
			Value: "NS2",
		})
	})

	t.Run("no modification when env var already has correct value", func(t *testing.T) {
		container := &corev1.Container{
			Name: "test",
			Env: []corev1.EnvVar{
				{Name: apiv1.EnvKymaLandscape, Value: "NS2"},
			},
		}
		modified := setLandscapeEnvVar(container, container.Env, "NS2")
		assert.False(t, modified)
		assert.Len(t, container.Env, 1)
	})

	t.Run("replaces env var when value differs", func(t *testing.T) {
		container := &corev1.Container{
			Name: "test",
			Env: []corev1.EnvVar{
				{Name: apiv1.EnvKymaLandscape, Value: "OLD"},
			},
		}
		modified := setLandscapeEnvVar(container, container.Env, "NS2")
		assert.True(t, modified)
		assert.Equal(t, "NS2", container.Env[0].Value)
	})
}

func TestBuildDefaulterSetLandscape(t *testing.T) {
	makePod := func(annotations map[string]string) *corev1.Pod {
		return &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: annotations,
			},
			Spec: corev1.PodSpec{
				InitContainers: []corev1.Container{
					{Name: "init", Image: "busybox"},
				},
				Containers: []corev1.Container{
					{Name: "main", Image: "nginx"},
				},
			},
		}
	}

	cfg := &apiv1.Config{
		Overrides: map[string]string{"x": "y"},
	}

	nsAnnotations := map[string]string{}

	t.Run("injects env var when annotation present and landscape non-empty", func(t *testing.T) {
		defaulter := BuildDefaulterSetLandscape("NS2")
		pod := makePod(map[string]string{apiv1.AnnotationSetLandscape: "true"})

		modified, err := defaulter(pod, nsAnnotations, cfg)
		assert.NoError(t, err)
		assert.True(t, modified)

		for _, c := range append(pod.Spec.InitContainers, pod.Spec.Containers...) {
			assert.Contains(t, c.Env, corev1.EnvVar{
				Name:  apiv1.EnvKymaLandscape,
				Value: "NS2",
			}, "container %s should have KYMA_LANDSCAPE", c.Name)
		}
	})

	t.Run("no injection when annotation absent", func(t *testing.T) {
		defaulter := BuildDefaulterSetLandscape("NS2")
		pod := makePod(nil)

		modified, err := defaulter(pod, nsAnnotations, cfg)
		assert.NoError(t, err)
		assert.False(t, modified)

		for _, c := range append(pod.Spec.InitContainers, pod.Spec.Containers...) {
			assert.Empty(t, c.Env, "container %s should have no env vars", c.Name)
		}
	})

	t.Run("injection via namespace annotation", func(t *testing.T) {
		defaulter := BuildDefaulterSetLandscape("CN")
		pod := makePod(nil)
		nsAnns := map[string]string{apiv1.AnnotationSetLandscape: "true"}

		modified, err := defaulter(pod, nsAnns, cfg)
		assert.NoError(t, err)
		assert.True(t, modified)

		assert.Contains(t, pod.Spec.Containers[0].Env, corev1.EnvVar{
			Name:  apiv1.EnvKymaLandscape,
			Value: "CN",
		})
	})
}
