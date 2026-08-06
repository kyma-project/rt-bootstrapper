package ctb

import (
	"context"
	"log/slog"
	"time"

	certificatesv1beta1 "k8s.io/api/certificates/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// +kubebuilder:rbac:groups="certificates.k8s.io",resources=clustertrustbundles,verbs=get;list;watch

// CTBWatcher watches a named ClusterTrustBundle and updates the hash holder on changes.
type CTBWatcher struct {
	client.Client
	Scheme     *runtime.Scheme
	CTBName    string
	HashHolder *HashHolder
}

func (w *CTBWatcher) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	if req.Name != w.CTBName {
		return ctrl.Result{}, nil
	}

	log := slog.Default().With("controller", "ctb-watcher", "ctb-name", req.Name)

	var ctb certificatesv1beta1.ClusterTrustBundle
	if err := w.Get(ctx, req.NamespacedName, &ctb); err != nil {
		log.Error("failed to get ClusterTrustBundle", "error", err)
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	oldHash := w.HashHolder.Get()
	w.HashHolder.ComputeAndSet(ctb.Spec.TrustBundle)
	newHash := w.HashHolder.Get()

	if oldHash != newHash {
		log.Info("CTB hash updated", "old", oldHash, "new", newHash)
	}

	requeue, err := RestartStalePods(ctx, w.Client, newHash)
	if err != nil {
		return ctrl.Result{}, err
	}
	if requeue {
		return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
	}

	return ctrl.Result{}, nil
}

func (w *CTBWatcher) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&certificatesv1beta1.ClusterTrustBundle{}).
		WithEventFilter(predicate.NewPredicateFuncs(func(object client.Object) bool {
			return object.GetName() == w.CTBName
		})).
		Named("ctb-watcher").
		Complete(w)
}

// PreComputeHash reads the named CTB and initializes the hash holder.
// Called at startup before the manager starts.
func PreComputeHash(ctx context.Context, c client.Client, ctbName string, holder *HashHolder) error {
	var ctb certificatesv1beta1.ClusterTrustBundle
	if err := c.Get(ctx, client.ObjectKey{Name: ctbName}, &ctb); err != nil {
		return err
	}
	holder.ComputeAndSet(ctb.Spec.TrustBundle)
	slog.Info("CTB hash pre-computed", "ctb-name", ctbName, "hash", holder.Get())
	return nil
}
