package ctb_test

import (
	"context"
	"testing"

	"github.com/kyma-project/rt-bootstrapper/internal/ctb"
	apiv1 "github.com/kyma-project/rt-bootstrapper/pkg/api/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func coreScheme(t *testing.T) *runtime.Scheme {
	t.Helper()
	s := runtime.NewScheme()
	require.NoError(t, clientgoscheme.AddToScheme(s))
	return s
}

func TestRestartStalePods_DeletesStalePodsOnly(t *testing.T) {
	s := coreScheme(t)

	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "kyma-system"}}
	stalePod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "stale-pod",
			Namespace: "kyma-system",
			Annotations: map[string]string{
				apiv1.AnnotationAddClusterTrustBundle: "restart-on-change",
				apiv1.AnnotationCTBHash:              "old-hash",
			},
		},
		Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "c", Image: "img"}}},
	}
	freshPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "fresh-pod",
			Namespace: "kyma-system",
			Annotations: map[string]string{
				apiv1.AnnotationAddClusterTrustBundle: "restart-on-change",
				apiv1.AnnotationCTBHash:              "new-hash",
			},
		},
		Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "c", Image: "img"}}},
	}
	truePod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "true-pod",
			Namespace: "kyma-system",
			Annotations: map[string]string{
				apiv1.AnnotationAddClusterTrustBundle: "true",
				apiv1.AnnotationCTBHash:              "old-hash",
			},
		},
		Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "c", Image: "img"}}},
	}

	fc := fake.NewClientBuilder().WithScheme(s).WithObjects(ns, stalePod, freshPod, truePod).Build()

	requeue, err := ctb.RestartStalePods(context.Background(), fc, "new-hash")
	require.NoError(t, err)
	assert.False(t, requeue)

	// stalePod should be deleted
	var pods corev1.PodList
	require.NoError(t, fc.List(context.Background(), &pods))
	names := []string{}
	for _, p := range pods.Items {
		names = append(names, p.Name)
	}
	assert.NotContains(t, names, "stale-pod")
	assert.Contains(t, names, "fresh-pod")
	assert.Contains(t, names, "true-pod")
}

func TestRestartStalePods_EmptyHashTreatedAsMatching(t *testing.T) {
	s := coreScheme(t)

	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "kyma-system"}}
	podNoHash := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "no-hash-pod",
			Namespace: "kyma-system",
			Annotations: map[string]string{
				apiv1.AnnotationAddClusterTrustBundle: "restart-on-change",
			},
		},
		Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "c", Image: "img"}}},
	}

	fc := fake.NewClientBuilder().WithScheme(s).WithObjects(ns, podNoHash).Build()

	requeue, err := ctb.RestartStalePods(context.Background(), fc, "desired-hash")
	require.NoError(t, err)
	assert.False(t, requeue)

	// pod with empty hash should NOT be deleted
	var pods corev1.PodList
	require.NoError(t, fc.List(context.Background(), &pods))
	assert.Len(t, pods.Items, 1)
}

func TestRestartStalePods_NoPodsNoRequeue(t *testing.T) {
	s := coreScheme(t)
	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "default"}}
	fc := fake.NewClientBuilder().WithScheme(s).WithObjects(ns).Build()

	requeue, err := ctb.RestartStalePods(context.Background(), fc, "hash")
	require.NoError(t, err)
	assert.False(t, requeue)
}
