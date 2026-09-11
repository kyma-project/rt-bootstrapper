## Context

The CTB pod restarter (`RestartStalePods`) iterates over every namespace in the cluster and attempts to list pods. Pod-list RBAC is granted through namespace-scoped `RoleBinding`s that reference the `pod-restarter-role` ClusterRole. Only managed namespaces — those declared as keys in the `namespaceFeatures` config — have these bindings. Listing pods in any other namespace returns a Forbidden error, which is currently logged uniformly at `Warn` level. This makes it impossible to tell expected permission denials apart from genuine RBAC misconfigurations on namespaces where RTB must operate.

## Goals / Non-Goals

**Goals:**
- Provide clear, actionable log output: errors for problems that need attention, warnings for expected conditions.
- Keep existing RBAC boundaries unchanged — no new cluster-scope permissions.
- Minimal, localised change to `restarter.go` and its call chain.

**Non-Goals:**
- Restricting the namespace iteration to only managed namespaces (pods can also be opted-in individually via pod-level annotations in any namespace that has a RoleBinding).
- Changing the RBAC model or adding a ClusterRoleBinding for pods.

## Decisions

### Pass managed namespace set through the call chain
The set of managed namespace names (keys of `NamespaceFeatures`) is built once at startup in `cmd/main.go` and stored on `CTBWatcher.ManagedNamespaces`. It is forwarded as a `map[string]bool` parameter to `RestartStalePods`.

**Rationale:** This avoids coupling the restarter to the config reader, keeps the function easily testable with any arbitrary set, and requires no additional API calls at reconciliation time.

**Alternative considered:** Re-read `NamespaceFeatures` from the ConfigMap on every reconciliation — rejected because it adds latency and failure modes for a value that is static for the lifetime of the process.

### Log level differentiation
- Forbidden on a managed namespace → `slog.Error` with the full error, so it surfaces in alerts and dashboards.
- Forbidden on an unmanaged namespace → `slog.Warn` without the full error, keeping logs tidy.

Both cases still `continue` (skip the namespace). No change to control flow.

## Risks / Trade-offs

- [Namespace added to `namespaceFeatures` at runtime but RoleBinding not yet created] → Restarter logs an error until RBAC catches up. This is the desired behaviour — it highlights a real gap.
- [`managedNamespaces` is a snapshot from startup] → If the ConfigMap is updated, the set is stale until the process restarts. Acceptable because `namespaceFeatures` changes are rare and require a rollout anyway.
