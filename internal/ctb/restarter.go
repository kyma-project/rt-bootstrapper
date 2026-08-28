package ctb

import (
	"context"
	"log/slog"

	apiv1 "github.com/kyma-project/rt-bootstrapper/pkg/api/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// RestartStalePods scans all namespaces for pods with annotation value "true"
// whose CTB hash doesn't match desiredHash, and deletes them.
// Pods without ownerReferences are skipped (orphan protection).
// Returns true if any pods were deleted (requeue needed).
func RestartStalePods(ctx context.Context, c client.Client, desiredHash string) (bool, error) {
	log := slog.Default().With("controller", "ctb-restarter", "desiredHash", desiredHash)

	var namespaces corev1.NamespaceList
	if err := c.List(ctx, &namespaces); err != nil {
		return false, err
	}

	var anyDeleted bool
	for _, ns := range namespaces.Items {
		deleted, err := restartStalePodsInNamespace(ctx, c, ns.Name, desiredHash, log)
		if err != nil {
			if errors.IsForbidden(err) {
				log.Warn("no permission to list pods, skipping namespace", "namespace", ns.Name)
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
