## Why

Pods that have the CTB restart annotation (`rt-cfg.kyma-project.io/add-cluster-trust-bundle: "true"`) but were created before the webhook stamped the `ctb-hash` annotation (or where the hash was not set for any reason) are silently skipped by `RestartStalePods`. The current logic compares `pod.Annotations[AnnotationCTBHash]` to `desiredHash`—when the annotation is missing the value is `""`, which never equals a valid SHA-256 hash, yet the comparison `podHash == desiredHash` still evaluates to `false`, so the pod **would** be deleted. However, the earlier guard `CTBRestartEnabled` only returns `true` when the annotation value is exactly `"true"`, and pods that received the CTB volume via namespace-level defaults (not pod-level annotation) may not carry the pod annotation at all—causing them to be skipped entirely. This means a class of opted-in pods is never restarted when the trust bundle rotates, leaving them with stale CA certificates.

## What Changes

- **Broaden restart eligibility in `RestartStalePods`**: Pods that received the CTB volume (i.e., were mutated by the webhook) should be eligible for restart even if they don't carry the pod-level `add-cluster-trust-bundle: "true"` annotation. The `AnnotationModified` annotation or the presence of the CTB volume itself can serve as the eligibility signal.
- **Treat missing hash as stale**: A pod that has the CTB volume but no `ctb-hash` annotation should be treated as stale and restarted, rather than silently skipped.
- **Update tests**: Add test cases covering pods without the hash annotation and pods opted-in via namespace defaults.

## Capabilities

### New Capabilities
- `ctb-restart-eligibility`: Rules for determining which pods are eligible for restart when the cluster trust bundle changes, including pods without the hash annotation.

### Modified Capabilities

## Impact

- `internal/ctb/restarter.go` – restart eligibility logic changes
- `internal/ctb/restarter_test.go` – new and updated test cases
- `pkg/api/v1/ctb_values.go` – possible new helper or adjustment to existing helpers
- `pkg/api/v1/ctb_values_test.go` – corresponding test updates
