## Why

When the ClusterTrustBundle (CTB) feature is activated via a **namespace** annotation (`rt-cfg.kyma-project.io/add-cluster-trust-bundle: "true"`) or via namespace default features in config, the webhook correctly mounts the CTB volume and stamps the `ctb-hash` annotation on newly created pods. However, when the ClusterTrustBundle later changes, the CTB restarter **does not restart those pods**. This is because `RestartStalePods` checks `CTBRestartEnabled(pod.Annotations)`, which looks for `rt-cfg.kyma-project.io/add-cluster-trust-bundle: "true"` on the pod's own annotations — but the webhook never copies the `add-cluster-trust-bundle` annotation onto the pod. Pods opted-in via namespace-level mechanisms silently keep stale CA certificates after a trust bundle rotation.

## What Changes

- **Change restart eligibility in `RestartStalePods`**: Use the presence of the `rt-bootstrapper.kyma-project.io/ctb-hash` annotation as the restart eligibility signal instead of `CTBRestartEnabled(pod.Annotations)`. Since the webhook stamps `ctb-hash` on every CTB-opted-in pod regardless of opt-in source (pod annotation, namespace annotation, or namespace defaults), this annotation is a reliable indicator that the pod participates in CTB and should be restarted when the bundle changes.
- **Update `CTBRestartEnabled` or introduce a new eligibility helper**: Replace or augment the existing check so that any pod carrying a `ctb-hash` annotation is considered restart-eligible.
- **Update tests**: Add test cases covering pods opted-in via namespace annotation/defaults that lack the pod-level `add-cluster-trust-bundle` annotation but carry `ctb-hash`.

## Capabilities

### New Capabilities
- `ctb-restart-eligibility`: Defines which pods are eligible for restart when the ClusterTrustBundle changes, ensuring pods opted-in via namespace-level annotations or defaults are included.

### Modified Capabilities

## Impact

- `internal/ctb/restarter.go` — restart eligibility logic changes
- `internal/ctb/restarter_test.go` — new and updated test cases
- `pkg/api/v1/ctb_values.go` — new or updated helper for restart eligibility
- `pkg/api/v1/ctb_values_test.go` — corresponding test updates
