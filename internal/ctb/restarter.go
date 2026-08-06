package ctb

import (
	"context"
	"log/slog"

	apiv1 "github.com/kyma-project/rt-bootstrapper/pkg/api/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// RestartStalePods scans all namespaces for pods with "restart-on-change" annotation
// whose CTB hash doesn't match desiredHash, and deletes them.
// Empty/missing hash on a pod is treated as matching (pod is skipped).
// Returns true if any pods were deleted (requeue needed).
func RestartStalePods(ctx context.Context, c client.Client, desiredHash string) (bool, error) {
	log := slog.Default().With("controller", "ctb-restarter", "desiredHash", desiredHash)

	var namespaces corev1.NamespaceList
	if err := c.List(ctx, &namespaces); err != nil {
		return false, err
	}

	for _, ns := range namespaces.Items {
		if err := restartStalePodsInNamespace(ctx, c, ns.Name, desiredHash, log); err != nil {
			if errors.IsForbidden(err) {
				log.Warn("no permission to list pods, skipping namespace", "namespace", ns.Name)
				continue
			}
			return false, err
		}
	}

	// ponytail: always false — caller requeueing on deletion not yet wired up
	return false, nil
}

func restartStalePodsInNamespace(ctx context.Context, c client.Client, namespace, desiredHash string, log *slog.Logger) error {
	var pods corev1.PodList
	if err := c.List(ctx, &pods, client.InNamespace(namespace)); err != nil {
		return err
	}

	for i := range pods.Items {
		pod := &pods.Items[i]

		if !apiv1.CTBRestartEnabled(pod.Annotations) {
			continue
		}

		// ponytail: empty hash = treat as matching; avoids deleting pods created before this feature
		podHash := pod.Annotations[apiv1.AnnotationCTBHash]
		if podHash == "" || podHash == desiredHash {
			continue
		}

		log.Info("deleting stale pod", "namespace", namespace, "pod", pod.Name, "podHash", podHash)
		if err := c.Delete(ctx, pod); err != nil && !errors.IsNotFound(err) {
			return err
		}
	}

	return nil
}
