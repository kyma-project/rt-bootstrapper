package ctb_test

import (
	"context"
	"testing"

	"github.com/kyma-project/rt-bootstrapper/internal/ctb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	certificatesv1alpha1 "k8s.io/api/certificates/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func newScheme(t *testing.T) *runtime.Scheme {
	t.Helper()
	s := runtime.NewScheme()
	require.NoError(t, certificatesv1alpha1.AddToScheme(s))
	return s
}

func TestCTBWatcher_Reconcile(t *testing.T) {
	s := newScheme(t)
	ctbObj := &certificatesv1alpha1.ClusterTrustBundle{
		ObjectMeta: metav1.ObjectMeta{Name: "my-ctb"},
		Spec:       certificatesv1alpha1.ClusterTrustBundleSpec{TrustBundle: "bundle-content"},
	}

	fakeClient := fake.NewClientBuilder().WithScheme(s).WithObjects(ctbObj).Build()
	holder := ctb.NewHashHolder()

	watcher := &ctb.CTBWatcher{
		Client:     fakeClient,
		Scheme:     s,
		CTBName:    "my-ctb",
		HashHolder: holder,
	}

	_, err := watcher.Reconcile(context.Background(), ctrl.Request{NamespacedName: types.NamespacedName{Name: "my-ctb"}})
	require.NoError(t, err)
	assert.NotEmpty(t, holder.Get())
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
	ctbObj := &certificatesv1alpha1.ClusterTrustBundle{
		ObjectMeta: metav1.ObjectMeta{Name: "my-ctb"},
		Spec:       certificatesv1alpha1.ClusterTrustBundleSpec{TrustBundle: "v1"},
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
	ctbObj := &certificatesv1alpha1.ClusterTrustBundle{
		ObjectMeta: metav1.ObjectMeta{Name: "my-ctb"},
		Spec:       certificatesv1alpha1.ClusterTrustBundleSpec{TrustBundle: "test-bundle"},
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
