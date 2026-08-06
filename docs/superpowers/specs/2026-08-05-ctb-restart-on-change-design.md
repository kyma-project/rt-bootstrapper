# Design: ClusterTrustBundle Restart-on-Change

## Progress

| PR | Scope | Status | Link |
|----|-------|--------|------|
| PR1 | Annotation value support (`false`, `restart-on-change`) | ✅ Done | [#190](https://github.com/kyma-project/rt-bootstrapper/pull/190) |
| PR2 | Hash stamping + CTB watcher | ⬜ Not started | — |
| PR3 | CTB controller + pod restart | ⬜ Not started | — |
| PR4 | RBAC (ClusterRole + RoleBindings) | ⬜ Not started | — |
| PR5 | Documentation update | ⬜ Not started | — |

## Summary

Automatically restart pods in Kyma-managed namespaces when the CA certificate in a `ClusterTrustBundle` (CTB) changes. Opt-in via annotation value `"restart-on-change"`.

## Annotation Values

| Value | Mount CTB | Restart by controller |
|-------|-----------|----------------------|
| `"false"` | ✗ | ✗ |
| `"true"` | ✓ | ✗ |
| `"restart-on-change"` | ✓ | ✓ |

- `"false"` — explicit opt-out. Needed for workloads in namespaces with default features that don't want CTB.
- `"true"` — existing behavior, mount only. Application handles cert refresh itself.
- `"restart-on-change"` — mount + controller-managed restart on CTB rotation.

## Architecture

### Components

1. **Webhook (existing, extended)** — on pod creation:
   - `"false"` → skip CTB mount entirely.
   - `"true"` → mount CTB projected volume (existing behavior).
   - `"restart-on-change"` → mount CTB projected volume + stamp hash annotation.

2. **CTB Watcher (new)** — watches the single named `ClusterTrustBundle` (name from existing `clusterTrustBundle.name` config). On change: compute SHA-256 hash of `.spec.trustBundle`, store in shared in-memory field.

3. **CTB Restart Controller (new)** — triggered by CTB change:
   - List all namespaces (cluster-wide permission already exists).
   - For each namespace, list pods. On 403 → log warning, skip.
   - For each pod with annotation `"restart-on-change"` and hash ≠ desired → delete pod.
   - Requeue until all matching pods have correct hash.

### Shared State

The CTB hash is held in memory (e.g., `sync.Mutex`-protected string or `atomic.Value`). Same process hosts both webhook and controller — no external state needed.

**Startup initialization:** The CTB is read and hashed once at startup (before the manager starts), so the shared field is never empty. This eliminates the race condition where the webhook could serve a pod admission before the controller's informer syncs. The controller's watch keeps the hash updated afterward.

### Data Flow

```
CTB changes
  → Controller computes hash, stores in shared field
  → Controller lists pods across namespaces (403 = warn + skip)
  → Pods with "restart-on-change" + stale hash → delete
  → Workload controller recreates pod
  → Webhook intercepts new pod creation
  → Webhook mounts CTB volume + stamps current hash from shared field
  → Controller requeues, sees matching hash → done
```

## Restart Mechanism

- **Delete the pod directly.** The owning workload controller (Deployment/DaemonSet/StatefulSet) recreates it.
- Webhook re-mutates the new pod on creation (mounts fresh CTB + stamps current hash).
- All matching pods deleted at once — Kyma system pods, brief disruption acceptable.

## Hash Annotation

- Key: `rt-bootstrapper.kyma-project.io/ctb-hash`
- Value: hex-encoded SHA-256 of the CTB `.spec.trustBundle` content.
- Stamped by the webhook on pods with `"restart-on-change"` only.
- Used by the controller for convergence checking.

## Empty Hash Semantics

- Empty/missing hash on a pod = **treat as matching** (don't restart).
- Only occurs on pods created before this feature was deployed (upgrade scenario).
- Startup race eliminated by pre-computing hash at startup (shared field is never empty in steady state).
- Avoids mass restarts on feature rollout — existing pods survive and get restarted on the next CTB rotation.

## Namespace Discovery & RBAC

- Controller iterates all namespaces. RBAC is the boundary.
- On 403 (no permission to list pods in a namespace) → log warning, skip.
- rt-bootstrapper ships:
  - A **ClusterRole** (generated via kubebuilder marker) with `list`, `delete` on pods.
  - **RoleBindings** in core namespaces (`kyma-system`, `istio-system`).
- Other Kyma component owners extend coverage by adding their own RoleBinding referencing the same ClusterRole.

## Backward Compatibility

- Annotation absent or `"true"` → no behavior change.
- Existing pods without hash annotation → not restarted (empty = matches).
- `"false"` is a new explicit opt-out for workloads in namespaces with default features.

## Incremental Delivery (PR Breakdown)

1. **PR1: Annotation value support** — webhook accepts `"false"` (opt-out) and `"restart-on-change"` (mount CTB, behaves like `"true"` for now). Tests.
2. **PR2: Hash stamping** — webhook stamps `rt-bootstrapper.kyma-project.io/ctb-hash` on pods with `"restart-on-change"`. Shared hash holder + CTB watcher (computes hash, no restart yet). Tests.
3. **PR3: CTB controller + pod restart** — controller reconciliation: list pods, compare hashes, delete stale pods, requeue until convergence. Tests.
4. **PR4: RBAC** — ClusterRole + RoleBindings for core namespaces. Wiring in `cmd/main.go`.
5. **PR5: Documentation** — update Pod Manipulations docs with new annotation values and restart semantics.

## Testing

Extend existing Ginkgo/Gomega suite:

**Controller tests:**
- CTB change → pods with `"restart-on-change"` + stale hash → deleted.
- Pods with matching hash → not deleted.
- Pods with `"true"` → never deleted.
- Missing/empty hash → treated as matching, no deletion.
- Namespace with 403 → warning logged, skipped, no error.
- Requeue until convergence.

**Webhook tests:**
- `"restart-on-change"` → CTB mounted + hash stamped.
- `"true"` → CTB mounted, no hash.
- `"false"` → no CTB mount, no hash.

## Related

- [Issue #178](https://github.com/kyma-project/rt-bootstrapper/issues/178)
- [kyma/backlog#9692](https://github.tools.sap/kyma/backlog/issues/9692) — Incident: NS2/Sovereign Cloud CA bundle rotation
- [Pod Manipulations docs](https://github.com/kyma-project/rt-bootstrapper/blob/main/docs/contributor/02-01-pod-manipulations.md)
