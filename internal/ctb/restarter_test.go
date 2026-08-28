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
				apiv1.AnnotationAddClusterTrustBundle: "true",
				apiv1.AnnotationCTBHash:              "old-hash",
			},
			OwnerReferences: []metav1.OwnerReference{{Name: "deploy", Kind: "ReplicaSet", APIVersion: "apps/v1", UID: "uid1"}},
		},
		Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "c", Image: "img"}}},
	}
	freshPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "fresh-pod",
			Namespace: "kyma-system",
			Annotations: map[string]string{
				apiv1.AnnotationAddClusterTrustBundle: "true",
				apiv1.AnnotationCTBHash:              "new-hash",
			},
			OwnerReferences: []metav1.OwnerReference{{Name: "deploy", Kind: "ReplicaSet", APIVersion: "apps/v1", UID: "uid2"}},
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
	assert.True(t, requeue)

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

func TestRestartStalePods_OrphanPodSkipped(t *testing.T) {
	s := coreScheme(t)

	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "kyma-system"}}
	orphanPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "orphan-pod",
			Namespace: "kyma-system",
			Annotations: map[string]string{
				apiv1.AnnotationAddClusterTrustBundle: "true",
				apiv1.AnnotationCTBHash:              "old-hash",
			},
			// No OwnerReferences — orphan pod
		},
		Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "c", Image: "img"}}},
	}

	fc := fake.NewClientBuilder().WithScheme(s).WithObjects(ns, orphanPod).Build()

	requeue, err := ctb.RestartStalePods(context.Background(), fc, "new-hash")
	require.NoError(t, err)
	assert.False(t, requeue)

	// orphan pod must NOT be deleted
	var pods corev1.PodList
	require.NoError(t, fc.List(context.Background(), &pods))
	assert.Len(t, pods.Items, 1)
	assert.Equal(t, "orphan-pod", pods.Items[0].Name)
}

func TestRestartStalePods_NoPodsNoRequeue(t *testing.T) {
	s := coreScheme(t)
	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "default"}}
	fc := fake.NewClientBuilder().WithScheme(s).WithObjects(ns).Build()

	requeue, err := ctb.RestartStalePods(context.Background(), fc, "hash")
	require.NoError(t, err)
	assert.False(t, requeue)
}

func TestRestartStalePods_MissingHashTreatedAsStale(t *testing.T) {
	s := coreScheme(t)

	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "kyma-system"}}
	// Pod has add-cluster-trust-bundle: "true" but NO ctb-hash annotation
	noHashPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "no-hash-pod",
			Namespace: "kyma-system",
			Annotations: map[string]string{
				apiv1.AnnotationAddClusterTrustBundle: "true",
			},
			OwnerReferences: []metav1.OwnerReference{{Name: "deploy", Kind: "ReplicaSet", APIVersion: "apps/v1", UID: "uid1"}},
		},
		Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "c", Image: "img"}}},
	}

	fc := fake.NewClientBuilder().WithScheme(s).WithObjects(ns, noHashPod).Build()

	requeue, err := ctb.RestartStalePods(context.Background(), fc, "new-hash")
	require.NoError(t, err)
	assert.True(t, requeue)

	// Pod should be deleted (missing hash treated as stale)
	var pods corev1.PodList
	require.NoError(t, fc.List(context.Background(), &pods))
	assert.Empty(t, pods.Items)
}

func TestRestartStalePods_CTBHashOnlyNotEligible(t *testing.T) {
	s := coreScheme(t)

	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "kyma-system"}}
	// Pod has ctb-hash but NOT add-cluster-trust-bundle annotation
	hashOnlyPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "hash-only-pod",
			Namespace: "kyma-system",
			Annotations: map[string]string{
				apiv1.AnnotationCTBHash: "old-hash",
			},
			OwnerReferences: []metav1.OwnerReference{{Name: "deploy", Kind: "ReplicaSet", APIVersion: "apps/v1", UID: "uid1"}},
		},
		Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "c", Image: "img"}}},
	}

	fc := fake.NewClientBuilder().WithScheme(s).WithObjects(ns, hashOnlyPod).Build()

	requeue, err := ctb.RestartStalePods(context.Background(), fc, "new-hash")
	require.NoError(t, err)
	assert.False(t, requeue)

	// Pod should NOT be deleted (not eligible without add-cluster-trust-bundle)
	var pods corev1.PodList
	require.NoError(t, fc.List(context.Background(), &pods))
	assert.Len(t, pods.Items, 1)
	assert.Equal(t, "hash-only-pod", pods.Items[0].Name)
}

func TestRestartStalePods_MatchingHashNotDeleted(t *testing.T) {
	s := coreScheme(t)

	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "kyma-system"}}
	matchingPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "matching-pod",
			Namespace: "kyma-system",
			Annotations: map[string]string{
				apiv1.AnnotationAddClusterTrustBundle: "true",
				apiv1.AnnotationCTBHash:              "current-hash",
			},
			OwnerReferences: []metav1.OwnerReference{{Name: "deploy", Kind: "ReplicaSet", APIVersion: "apps/v1", UID: "uid1"}},
		},
		Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "c", Image: "img"}}},
	}

	fc := fake.NewClientBuilder().WithScheme(s).WithObjects(ns, matchingPod).Build()

	requeue, err := ctb.RestartStalePods(context.Background(), fc, "current-hash")
	require.NoError(t, err)
	assert.False(t, requeue)

	var pods corev1.PodList
	require.NoError(t, fc.List(context.Background(), &pods))
	assert.Len(t, pods.Items, 1)
	assert.Equal(t, "matching-pod", pods.Items[0].Name)
}

func TestRestartStalePods_OrphanWithCTBAnnotationNotDeleted(t *testing.T) {
	s := coreScheme(t)

	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "kyma-system"}}
	orphanPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "orphan-ctb-pod",
			Namespace: "kyma-system",
			Annotations: map[string]string{
				apiv1.AnnotationAddClusterTrustBundle: "true",
			},
			// No OwnerReferences — orphan pod
		},
		Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "c", Image: "img"}}},
	}

	fc := fake.NewClientBuilder().WithScheme(s).WithObjects(ns, orphanPod).Build()

	requeue, err := ctb.RestartStalePods(context.Background(), fc, "new-hash")
	require.NoError(t, err)
	assert.False(t, requeue)

	// Orphan pod must NOT be deleted
	var pods corev1.PodList
	require.NoError(t, fc.List(context.Background(), &pods))
	assert.Len(t, pods.Items, 1)
	assert.Equal(t, "orphan-ctb-pod", pods.Items[0].Name)
}
