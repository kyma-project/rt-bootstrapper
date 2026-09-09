## 1. Core Implementation

- [x] 1.1 Add `managedNamespaces map[string]bool` parameter to `RestartStalePods` in `internal/ctb/restarter.go`
- [x] 1.2 Branch Forbidden-error handling: log `Error` for managed namespaces (with full error), log `Warn` for unmanaged namespaces
- [x] 1.3 Add `ManagedNamespaces map[string]bool` field to `CTBWatcher` in `internal/ctb/watcher.go`
- [x] 1.4 Pass `w.ManagedNamespaces` from `CTBWatcher.Reconcile` to `RestartStalePods`

## 2. Wiring

- [x] 2.1 In `cmd/main.go`, build `managedNS` set from `cfg.NamespaceFeatures` keys
- [x] 2.2 Pass `managedNS` to `CTBWatcher` initialization via the new `ManagedNamespaces` field

## 3. Tests

- [x] 3.1 Update all existing `RestartStalePods` call sites in `internal/ctb/restarter_test.go` to pass `nil` for the new parameter (preserves existing behaviour)
