# Remove `restart-on-change` annotation value

## Why

The ClusterTrustBundle annotation currently supports three values: `"true"`, `"false"`, and `"restart-on-change"`. This creates confusion because `"true"` only mounts the CTB volume without restarting pods, while `"restart-on-change"` both mounts and restarts. The dual-value semantics are inconsistent — users shouldn't need to know which of two truthy values to pick when they want full CTB functionality. Collapsing them into a single `"true"` that does both simplifies the mental model and reduces configuration errors.

This change also aligns with Kubernetes best practices: workloads should not break when a pod is restarted. By keeping CTB mounting and pod restart coupled under a single value, we enforce a predictable pattern — if a pod has a CTB, restarting it will always restore the correct bundle, avoiding subtle failures where a pod restarts with a missing or stale trust bundle.

## What Changes

- **Remove** the `"restart-on-change"` annotation value entirely. It becomes an invalid/ignored value.
- **Change** `"true"` to both mount the CTB volume **and** trigger pod restarts when the CTB hash changes (current `"restart-on-change"` behavior).
- **`"false"`** remains unchanged: neither mounts CTB nor restarts.
- All test code referencing `"restart-on-change"` is updated to use `"true"`.

## Capabilities

### New Capabilities
- `ctb-annotation-value`: Defines the simplified CTB annotation value semantics (only `"true"` and `"false"`).

### Modified Capabilities
<!-- None — no existing main specs exist yet. This creates a new capability. -->

## Impact

| Area | Files affected |
|------|---------------|
| API constants | `pkg/api/v1/ctb_values.go` — remove `CTBValueRestartOnChange` |
| CTB defaulting | `internal/webhook/v1/pod_defaulters.go` — check `"true"` instead of `"restart-on-change"` for hash stamping |
| CTB restarter | `internal/ctb/restarter.go` — call simplified check for restart eligibility |
| Watcher tests | `internal/ctb/watcher_test.go` |
| Restarter tests | `internal/ctb/restarter_test.go` |
| Webhook tests | `internal/webhook/v1/pod_webhook_test.go` |
| Types tests | `pkg/api/v1/types_test.go` |
