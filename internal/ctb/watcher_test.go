package ctb_test

import (
	"context"
	"testing"
	"time"

	"github.com/kyma-project/rt-bootstrapper/internal/ctb"
	apiv1 "github.com/kyma-project/rt-bootstrapper/pkg/api/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	certificatesv1beta1 "k8s.io/api/certificates/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func newScheme(t *testing.T) *runtime.Scheme {
	t.Helper()
	s := runtime.NewScheme()
	require.NoError(t, certificatesv1beta1.AddToScheme(s))
	require.NoError(t, clientgoscheme.AddToScheme(s))
	return s
}

func TestCTBWatcher_Reconcile(t *testing.T) {
	s := newScheme(t)
	ctbObj := &certificatesv1beta1.ClusterTrustBundle{
		ObjectMeta: metav1.ObjectMeta{Name: "my-ctb"},
		Spec:       certificatesv1beta1.ClusterTrustBundleSpec{TrustBundle: "bundle-content"},
	}

	fakeClient := fake.NewClientBuilder().WithScheme(s).WithObjects(ctbObj).Build()
	holder := ctb.NewHashHolder()

	watcher := &ctb.CTBWatcher{
		Client:     fakeClient,
		Scheme:     s,
		CTBName:    "my-ctb",
		HashHolder: holder,
	}

	result, err := watcher.Reconcile(context.Background(), ctrl.Request{NamespacedName: types.NamespacedName{Name: "my-ctb"}})
	require.NoError(t, err)
	assert.NotEmpty(t, holder.Get())
	assert.Equal(t, 5*time.Minute, result.RequeueAfter)
}

func TestCTBWatcher_Reconcile_IgnoresOtherCTBs(t *testing.T) {
	s := newScheme(t)
	fakeClient := fake.NewClientBuilder().WithScheme(s).Build()
	holder := ctb.NewHashHolder()

	watcher := &ctb.CTBWatcher{
		Client:     fakeClient,
		Scheme:     s,
		CTBName:    "my-ctb",
		HashHolder: holder,
	}

	_, err := watcher.Reconcile(context.Background(), ctrl.Request{NamespacedName: types.NamespacedName{Name: "other-ctb"}})
	require.NoError(t, err)
	assert.Empty(t, holder.Get())
}

func TestCTBWatcher_Reconcile_HashChangesOnUpdate(t *testing.T) {
	s := newScheme(t)
	ctbObj := &certificatesv1beta1.ClusterTrustBundle{
		ObjectMeta: metav1.ObjectMeta{Name: "my-ctb"},
		Spec:       certificatesv1beta1.ClusterTrustBundleSpec{TrustBundle: "v1"},
	}

	fakeClient := fake.NewClientBuilder().WithScheme(s).WithObjects(ctbObj).Build()
	holder := ctb.NewHashHolder()

	watcher := &ctb.CTBWatcher{
		Client:     fakeClient,
		Scheme:     s,
		CTBName:    "my-ctb",
		HashHolder: holder,
	}

	req := ctrl.Request{NamespacedName: types.NamespacedName{Name: "my-ctb"}}
	_, err := watcher.Reconcile(context.Background(), req)
	require.NoError(t, err)
	hash1 := holder.Get()

	// Simulate content change
	ctbObj.Spec.TrustBundle = "v2"
	require.NoError(t, fakeClient.Update(context.Background(), ctbObj))

	_, err = watcher.Reconcile(context.Background(), req)
	require.NoError(t, err)
	hash2 := holder.Get()

	assert.NotEqual(t, hash1, hash2)
}

func TestPreComputeHash(t *testing.T) {
	s := newScheme(t)
	ctbObj := &certificatesv1beta1.ClusterTrustBundle{
		ObjectMeta: metav1.ObjectMeta{Name: "my-ctb"},
		Spec:       certificatesv1beta1.ClusterTrustBundleSpec{TrustBundle: "test-bundle"},
	}

	fakeClient := fake.NewClientBuilder().WithScheme(s).WithObjects(ctbObj).Build()
	holder := ctb.NewHashHolder()

	require.NoError(t, ctb.PreComputeHash(context.Background(), fakeClient, "my-ctb", holder))
	assert.NotEmpty(t, holder.Get())
}

func TestPreComputeHash_NotFound(t *testing.T) {
	s := newScheme(t)
	fakeClient := fake.NewClientBuilder().WithScheme(s).Build()
	holder := ctb.NewHashHolder()

	err := ctb.PreComputeHash(context.Background(), fakeClient, "missing-ctb", holder)
	require.Error(t, err)
	assert.Empty(t, holder.Get())
}

func TestCTBWatcher_Reconcile_TriggersRestart(t *testing.T) {
	s := newScheme(t)
	require.NoError(t, clientgoscheme.AddToScheme(s))

	ctbObj := &certificatesv1beta1.ClusterTrustBundle{
		ObjectMeta: metav1.ObjectMeta{Name: "my-ctb"},
		Spec:       certificatesv1beta1.ClusterTrustBundleSpec{TrustBundle: "new-bundle"},
	}
	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "kyma-system"}}
	stalePod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "stale",
			Namespace: "kyma-system",
			Annotations: map[string]string{
				apiv1.AnnotationAddClusterTrustBundle: "restart-on-change",
				apiv1.AnnotationCTBHash:              "old-hash",
			},
			OwnerReferences: []metav1.OwnerReference{{Name: "deploy", Kind: "ReplicaSet", APIVersion: "apps/v1", UID: "uid1"}},
		},
		Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "c", Image: "img"}}},
	}

	fc := fake.NewClientBuilder().WithScheme(s).WithObjects(ctbObj, ns, stalePod).Build()
	holder := ctb.NewHashHolder()
	holder.Set("old-hash") // simulate previous state

	watcher := &ctb.CTBWatcher{
		Client:     fc,
		Scheme:     s,
		CTBName:    "my-ctb",
		HashHolder: holder,
	}

	req := ctrl.Request{NamespacedName: types.NamespacedName{Name: "my-ctb"}}
	result, err := watcher.Reconcile(context.Background(), req)
	require.NoError(t, err)

	// Should requeue since pods were deleted
	assert.True(t, result.RequeueAfter > 0)

	// Pod should be deleted
	var pods corev1.PodList
	require.NoError(t, fc.List(context.Background(), &pods))
	for _, p := range pods.Items {
		assert.NotEqual(t, "stale", p.Name)
	}
}
