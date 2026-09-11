package ctb

import (
	"context"
	"log/slog"

	apiv1 "github.com/kyma-project/rt-bootstrapper/pkg/api/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// RestartStalePods scans managed namespaces for pods with annotation value "true"
// whose CTB hash doesn't match desiredHash, and deletes them.
// Pods without ownerReferences are skipped (orphan protection).
// managedNamespaces is the set of namespace names where RTB is expected to
// have pod permissions (i.e. keys of namespaceFeatures). Only these namespaces
// are visited: the pod informer cache is restricted to the same set, so calling
// c.List for any other namespace would return "unknown namespace for the cache".
// A Forbidden error on a managed namespace is logged as an error (unexpected
// RBAC misconfiguration); other namespaces are silently skipped.
// Returns true if any pods were deleted (requeue needed).
func RestartStalePods(ctx context.Context, c client.Client, desiredHash string, managedNamespaces map[string]bool) (bool, error) {
	log := slog.Default().With("controller", "ctb-restarter", "desiredHash", desiredHash)

	var namespaces corev1.NamespaceList
	if err := c.List(ctx, &namespaces); err != nil {
		return false, err
	}

	var anyDeleted bool
	for _, ns := range namespaces.Items {
		// Skip namespaces outside the pod cache scope. The pod informer cache is
		// restricted to managedNamespaces (keys of namespaceFeatures); calling
		// c.List for any other namespace returns "unknown namespace for the cache".
		if !managedNamespaces[ns.Name] {
			continue
		}
		deleted, err := restartStalePodsInNamespace(ctx, c, ns.Name, desiredHash, log)
		if err != nil {
			if errors.IsForbidden(err) {
				log.Error("no permission to list pods in managed namespace", "namespace", ns.Name, "error", err)
				continue
			}
			return false, err
		}
		if deleted {
			anyDeleted = true
		}
	}

	return anyDeleted, nil
}

func restartStalePodsInNamespace(ctx context.Context, c client.Client, namespace, desiredHash string, log *slog.Logger) (bool, error) {
	var pods corev1.PodList
	if err := c.List(ctx, &pods, client.InNamespace(namespace)); err != nil {
		return false, err
	}

	var anyDeleted bool
	for i := range pods.Items {
		pod := &pods.Items[i]

		if !apiv1.CTBRestartEnabled(pod.Annotations) {
			continue
		}

		if len(pod.OwnerReferences) == 0 {
			log.Warn("orphan pod has CTB annotation but no owner, skipping", "namespace", namespace, "pod", pod.Name)
			continue
		}

		podHash := pod.Annotations[apiv1.AnnotationCTBHash]
		if podHash == desiredHash {
			continue
		}

		log.Info("deleting stale pod", "namespace", namespace, "pod", pod.Name, "podHash", podHash)
		if err := c.Delete(ctx, pod); err != nil && !errors.IsNotFound(err) {
			return false, err
		}
		anyDeleted = true
	}

	return anyDeleted, nil
}
