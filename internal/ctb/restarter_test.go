package ctb_test

import (
	"context"
	"log/slog"
	"sync"
	"testing"

	"github.com/kyma-project/rt-bootstrapper/internal/ctb"
	apiv1 "github.com/kyma-project/rt-bootstrapper/pkg/api/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
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
				apiv1.AnnotationCTBHash:               "old-hash",
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
				apiv1.AnnotationCTBHash:               "new-hash",
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
				apiv1.AnnotationCTBHash:               "old-hash",
			},
		},
		Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "c", Image: "img"}}},
	}

	fc := fake.NewClientBuilder().WithScheme(s).WithObjects(ns, stalePod, freshPod, truePod).Build()

	requeue, err := ctb.RestartStalePods(context.Background(), fc, "new-hash", map[string]bool{"kyma-system": true})
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
				apiv1.AnnotationCTBHash:               "old-hash",
			},
			// No OwnerReferences — orphan pod
		},
		Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "c", Image: "img"}}},
	}

	fc := fake.NewClientBuilder().WithScheme(s).WithObjects(ns, orphanPod).Build()

	requeue, err := ctb.RestartStalePods(context.Background(), fc, "new-hash", map[string]bool{"kyma-system": true})
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

	requeue, err := ctb.RestartStalePods(context.Background(), fc, "hash", map[string]bool{"kyma-system": true})
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

	requeue, err := ctb.RestartStalePods(context.Background(), fc, "new-hash", map[string]bool{"kyma-system": true})
	require.NoError(t, err)
	assert.True(t, requeue)

	// Pod should be deleted (missing hash treated as stale)
	var pods corev1.PodList
	require.NoError(t, fc.List(context.Background(), &pods))
	assert.Empty(t, pods.Items)
}

func TestRestartStalePods_CTBHashOnlyIsEligible(t *testing.T) {
	s := coreScheme(t)

	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "kyma-system"}}
	// Pod has ctb-hash but NOT add-cluster-trust-bundle annotation
	// (opted-in via namespace annotation — webhook stamped ctb-hash)
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

	requeue, err := ctb.RestartStalePods(context.Background(), fc, "new-hash", map[string]bool{"kyma-system": true})
	require.NoError(t, err)
	assert.True(t, requeue)

	// Pod should be deleted (ctb-hash present → eligible, stale hash → deleted)
	var pods corev1.PodList
	require.NoError(t, fc.List(context.Background(), &pods))
	assert.Empty(t, pods.Items)
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
				apiv1.AnnotationCTBHash:               "current-hash",
			},
			OwnerReferences: []metav1.OwnerReference{{Name: "deploy", Kind: "ReplicaSet", APIVersion: "apps/v1", UID: "uid1"}},
		},
		Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "c", Image: "img"}}},
	}

	fc := fake.NewClientBuilder().WithScheme(s).WithObjects(ns, matchingPod).Build()

	requeue, err := ctb.RestartStalePods(context.Background(), fc, "current-hash", map[string]bool{"kyma-system": true})
	require.NoError(t, err)
	assert.False(t, requeue)

	var pods corev1.PodList
	require.NoError(t, fc.List(context.Background(), &pods))
	assert.Len(t, pods.Items, 1)
	assert.Equal(t, "matching-pod", pods.Items[0].Name)
}

func TestRestartStalePods_CTBHashOnlyWithStaleHash_IsDeleted(t *testing.T) {
	s := coreScheme(t)

	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "kyma-system"}}
	// Pod opted-in via namespace annotation: webhook stamped ctb-hash but
	// the pod itself does NOT carry add-cluster-trust-bundle.
	nsPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ns-opted-pod",
			Namespace: "kyma-system",
			Annotations: map[string]string{
				apiv1.AnnotationCTBHash: "old-hash",
			},
			OwnerReferences: []metav1.OwnerReference{{Name: "deploy", Kind: "ReplicaSet", APIVersion: "apps/v1", UID: "uid1"}},
		},
		Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "c", Image: "img"}}},
	}

	fc := fake.NewClientBuilder().WithScheme(s).WithObjects(ns, nsPod).Build()

	requeue, err := ctb.RestartStalePods(context.Background(), fc, "new-hash", map[string]bool{"kyma-system": true})
	require.NoError(t, err)
	assert.True(t, requeue)

	// Pod should be deleted — ctb-hash is stale
	var pods corev1.PodList
	require.NoError(t, fc.List(context.Background(), &pods))
	assert.Empty(t, pods.Items)
}

func TestRestartStalePods_CTBHashOnlyMatchingHash_NotDeleted(t *testing.T) {
	s := coreScheme(t)

	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "kyma-system"}}
	// Pod opted-in via namespace: has ctb-hash matching desired hash
	matchingPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "matching-ns-pod",
			Namespace: "kyma-system",
			Annotations: map[string]string{
				apiv1.AnnotationCTBHash: "current-hash",
			},
			OwnerReferences: []metav1.OwnerReference{{Name: "deploy", Kind: "ReplicaSet", APIVersion: "apps/v1", UID: "uid1"}},
		},
		Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "c", Image: "img"}}},
	}

	fc := fake.NewClientBuilder().WithScheme(s).WithObjects(ns, matchingPod).Build()

	requeue, err := ctb.RestartStalePods(context.Background(), fc, "current-hash", map[string]bool{"kyma-system": true})
	require.NoError(t, err)
	assert.False(t, requeue)

	var pods corev1.PodList
	require.NoError(t, fc.List(context.Background(), &pods))
	assert.Len(t, pods.Items, 1)
	assert.Equal(t, "matching-ns-pod", pods.Items[0].Name)
}

func TestRestartStalePods_OrphanWithCTBHashNotDeleted(t *testing.T) {
	s := coreScheme(t)

	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "kyma-system"}}
	orphanPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "orphan-hash-pod",
			Namespace: "kyma-system",
			Annotations: map[string]string{
				apiv1.AnnotationCTBHash: "old-hash",
			},
			// No OwnerReferences — orphan pod
		},
		Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "c", Image: "img"}}},
	}

	fc := fake.NewClientBuilder().WithScheme(s).WithObjects(ns, orphanPod).Build()

	requeue, err := ctb.RestartStalePods(context.Background(), fc, "new-hash", map[string]bool{"kyma-system": true})
	require.NoError(t, err)
	assert.False(t, requeue)

	// Orphan pod must NOT be deleted
	var pods corev1.PodList
	require.NoError(t, fc.List(context.Background(), &pods))
	assert.Len(t, pods.Items, 1)
	assert.Equal(t, "orphan-hash-pod", pods.Items[0].Name)
}

func TestRestartStalePods_NoCTBAnnotations_NotDeleted(t *testing.T) {
	s := coreScheme(t)

	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "kyma-system"}}
	regularPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "regular-pod",
			Namespace: "kyma-system",
			Annotations: map[string]string{
				"app": "myapp",
			},
			OwnerReferences: []metav1.OwnerReference{{Name: "deploy", Kind: "ReplicaSet", APIVersion: "apps/v1", UID: "uid1"}},
		},
		Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "c", Image: "img"}}},
	}

	fc := fake.NewClientBuilder().WithScheme(s).WithObjects(ns, regularPod).Build()

	requeue, err := ctb.RestartStalePods(context.Background(), fc, "new-hash", map[string]bool{"kyma-system": true})
	require.NoError(t, err)
	assert.False(t, requeue)

	var pods corev1.PodList
	require.NoError(t, fc.List(context.Background(), &pods))
	assert.Len(t, pods.Items, 1)
	assert.Equal(t, "regular-pod", pods.Items[0].Name)
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

	requeue, err := ctb.RestartStalePods(context.Background(), fc, "new-hash", map[string]bool{"kyma-system": true})
	require.NoError(t, err)
	assert.False(t, requeue)

	// Orphan pod must NOT be deleted
	var pods corev1.PodList
	require.NoError(t, fc.List(context.Background(), &pods))
	assert.Len(t, pods.Items, 1)
	assert.Equal(t, "orphan-ctb-pod", pods.Items[0].Name)
}

// logRecord holds a captured slog record for test assertions.
type logRecord struct {
	Level   slog.Level
	Message string
	Attrs   map[string]string
}

// capturingHandler is an slog.Handler that records all log records.
type capturingHandler struct {
	mu      sync.Mutex
	records []logRecord
}

func (h *capturingHandler) Enabled(_ context.Context, _ slog.Level) bool { return true }

func (h *capturingHandler) Handle(_ context.Context, r slog.Record) error {
	attrs := make(map[string]string)
	r.Attrs(func(a slog.Attr) bool {
		attrs[a.Key] = a.Value.String()
		return true
	})
	h.mu.Lock()
	defer h.mu.Unlock()
	h.records = append(h.records, logRecord{Level: r.Level, Message: r.Message, Attrs: attrs})
	return nil
}

func (h *capturingHandler) WithAttrs(_ []slog.Attr) slog.Handler { return h }
func (h *capturingHandler) WithGroup(_ string) slog.Handler      { return h }

func (h *capturingHandler) findByMessage(msg string) []logRecord {
	h.mu.Lock()
	defer h.mu.Unlock()
	var out []logRecord
	for _, r := range h.records {
		if r.Message == msg {
			out = append(out, r)
		}
	}
	return out
}

// forbiddenListInterceptor returns an interceptor.Funcs that returns a
// Forbidden error when listing pods in any of the given namespaces.
func forbiddenListInterceptor(forbiddenNamespaces map[string]bool) interceptor.Funcs {
	return interceptor.Funcs{
		List: func(ctx context.Context, c client.WithWatch, list client.ObjectList, opts ...client.ListOption) error {
			listOpts := &client.ListOptions{}
			for _, o := range opts {
				o.ApplyToList(listOpts)
			}
			if _, ok := list.(*corev1.PodList); ok && forbiddenNamespaces[listOpts.Namespace] {
				return k8serrors.NewForbidden(
					schema.GroupResource{Resource: "pods"},
					"",
					k8serrors.NewUnauthorized("fake forbidden"),
				)
			}
			return c.List(ctx, list, opts...)
		},
	}
}

func TestRestartStalePods_ForbiddenOnManagedNamespace_LogsError(t *testing.T) {
	s := coreScheme(t)

	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "kyma-system"}}

	fc := fake.NewClientBuilder().
		WithScheme(s).
		WithObjects(ns).
		WithInterceptorFuncs(forbiddenListInterceptor(map[string]bool{"kyma-system": true})).
		Build()

	// Capture logs
	handler := &capturingHandler{}
	original := slog.Default()
	slog.SetDefault(slog.New(handler))
	defer slog.SetDefault(original)

	managedNS := map[string]bool{"kyma-system": true}
	requeue, err := ctb.RestartStalePods(context.Background(), fc, "hash", managedNS)
	require.NoError(t, err)
	assert.False(t, requeue)

	// Should log at Error level for managed namespace
	records := handler.findByMessage("no permission to list pods in managed namespace")
	require.Len(t, records, 1)
	assert.Equal(t, slog.LevelError, records[0].Level)
	assert.Equal(t, "kyma-system", records[0].Attrs["namespace"])
	assert.Contains(t, records[0].Attrs["error"], "forbidden")
}

func TestRestartStalePods_UnmanagedNamespace_Skipped(t *testing.T) {
	// Unmanaged namespaces are skipped entirely before any pod List call.
	// The interceptor would return Forbidden, but it must never be reached.
	s := coreScheme(t)

	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "customer-ns"}}

	fc := fake.NewClientBuilder().
		WithScheme(s).
		WithObjects(ns).
		WithInterceptorFuncs(forbiddenListInterceptor(map[string]bool{"customer-ns": true})).
		Build()

	handler := &capturingHandler{}
	original := slog.Default()
	slog.SetDefault(slog.New(handler))
	defer slog.SetDefault(original)

	// customer-ns is NOT in the managed set — it must be silently skipped
	managedNS := map[string]bool{"kyma-system": true}
	requeue, err := ctb.RestartStalePods(context.Background(), fc, "hash", managedNS)
	require.NoError(t, err)
	assert.False(t, requeue)

	// No log at any level for unmanaged namespace — it is skipped before the List
	assert.Empty(t, handler.findByMessage("no permission to list pods, skipping namespace"))
	assert.Empty(t, handler.findByMessage("no permission to list pods in managed namespace"))
}

func TestRestartStalePods_NilManagedNamespaces_AllNamespacesSkipped(t *testing.T) {
	// When managedNamespaces is nil, every namespace is unmanaged and skipped.
	s := coreScheme(t)

	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "some-ns"}}
	stalePod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "stale-pod",
			Namespace: "some-ns",
			Annotations: map[string]string{
				apiv1.AnnotationAddClusterTrustBundle: "true",
				apiv1.AnnotationCTBHash:               "old-hash",
			},
			OwnerReferences: []metav1.OwnerReference{{Name: "deploy", Kind: "ReplicaSet", APIVersion: "apps/v1", UID: "uid1"}},
		},
		Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "c", Image: "img"}}},
	}

	fc := fake.NewClientBuilder().
		WithScheme(s).
		WithObjects(ns, stalePod).
		Build()

	requeue, err := ctb.RestartStalePods(context.Background(), fc, "new-hash", nil)
	require.NoError(t, err)
	// nil managedNamespaces → every namespace skipped → no deletions
	assert.False(t, requeue)
}

func TestRestartStalePods_UnmanagedNamespaceSkipped_ManagedStillProcessed(t *testing.T) {
	// Unmanaged namespaces are skipped; managed ones are still processed.
	s := coreScheme(t)

	unmanagedNs := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "unmanaged-ns"}}
	kymaSystemNs := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "kyma-system"}}
	stalePod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "stale-pod",
			Namespace: "kyma-system",
			Annotations: map[string]string{
				apiv1.AnnotationAddClusterTrustBundle: "true",
				apiv1.AnnotationCTBHash:               "old-hash",
			},
			OwnerReferences: []metav1.OwnerReference{{Name: "deploy", Kind: "ReplicaSet", APIVersion: "apps/v1", UID: "uid1"}},
		},
		Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "c", Image: "img"}}},
	}

	// unmanaged-ns would return Forbidden — but it must never be reached
	fc := fake.NewClientBuilder().
		WithScheme(s).
		WithObjects(unmanagedNs, kymaSystemNs, stalePod).
		WithInterceptorFuncs(forbiddenListInterceptor(map[string]bool{"unmanaged-ns": true})).
		Build()

	managedNS := map[string]bool{"kyma-system": true}
	requeue, err := ctb.RestartStalePods(context.Background(), fc, "new-hash", managedNS)
	require.NoError(t, err)
	// Stale pod in kyma-system is deleted
	assert.True(t, requeue)

	var pods corev1.PodList
	require.NoError(t, fc.List(context.Background(), &pods))
	for _, p := range pods.Items {
		assert.NotEqual(t, "stale-pod", p.Name)
	}
}
