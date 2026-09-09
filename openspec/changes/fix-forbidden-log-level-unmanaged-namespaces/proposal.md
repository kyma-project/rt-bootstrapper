## Why

The CTB pod restarter iterates over all cluster namespaces and lists pods in each one. The pod-restarter RBAC is only bound via RoleBindings to managed namespaces (those listed as keys in `namespaceFeatures`), so Forbidden errors on all other namespaces are expected. Currently these are all logged at the same `Warn` level, making it impossible to distinguish an expected Forbidden (unmanaged namespace) from a genuine RBAC misconfiguration on a managed namespace.

## What Changes

- `RestartStalePods` accepts a set of managed namespace names (keys of `namespaceFeatures`) so it can distinguish managed from unmanaged namespaces.
- Forbidden errors on a **managed namespace** are logged at `Error` level with the full error — this signals a real RBAC problem.
- Forbidden errors on an **unmanaged namespace** remain at `Warn` level — expected and silently skipped.
- `CTBWatcher` carries the managed namespace set and passes it through to `RestartStalePods`.
- `cmd/main.go` builds the managed namespace set from `cfg.NamespaceFeatures` keys at startup.

## Capabilities

### New Capabilities

_(none)_

### Modified Capabilities

- `ctb-restart-eligibility`: Adds requirements for how Forbidden errors during pod listing are logged, differentiating between managed and unmanaged namespaces.

## Impact

- `internal/ctb/restarter.go` — signature change, new logging logic
- `internal/ctb/watcher.go` — new `ManagedNamespaces` field on `CTBWatcher`
- `cmd/main.go` — builds and passes managed namespace set
- `internal/ctb/restarter_test.go` — updated call sites for new parameter
