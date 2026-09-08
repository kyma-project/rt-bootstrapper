# Design: Remove `restart-on-change` annotation value

## Context

The ClusterTrustBundle (CTB) annotation (`apiv1.AnnotationAddClusterTrustBundle`) currently accepts three string values:

```
┌──────────────────────────────────────────────────────────┐
│  CURRENT TRIVARIATE VALUES                               │
├──────────────┬───────────────────────────────────────────┤
│  Value       │  Behavior                                 │
├──────────────┼───────────────────────────────────────────┤
│  "true"      │  Mount CTB volume only                  │
│  "false"     │  Do nothing (explicit opt-out)          │
│  "restart-  │  Mount CTB + restart pod on hash change │
│  -on-change" │                                         │
└──────────────┴───────────────────────────────────────────┘
```

The problem: two values (`"true"` and `"restart-on-change"`) both enable CTB mounting, but differ on whether pods get restarted. Users face a confusing choice. The intended "full CTB experience" (mount + restart) is only available via the verbose `"restart-on-change"`.

## Goals / Non-Goals

**Goals:**
- Collapse `"restart-on-change"` into `"true"`.
- `"true"` means: mount CTB **and** restart pods when CTB hash changes.
- `"false"` means: do nothing.
- Keep `"false"` opt-out semantics unchanged.

**Non-Goals:**
- Do not support a value that mounts CTB without restarting.
- Do not change the restarter logic (it already deletes stale pods correctly).
- Do not change the hash holder, watcher, or CRD definitions.

## Decisions

### D1: `"true"` does both mount + restart

**Decision:** The `"true"` annotation value will simultaneously mount the CTB volume and enable pod restart on hash changes.

**Rationale:**
- The vast majority of users who want CTB also want automatic restart when the bundle changes.
- Eliminating the "mount only" case removes an option that was only useful for very specific edge cases.
- Simpler API surface = fewer configuration mistakes.

**Alternatives considered:**
1. Keep `"true"` as mount-only and add `"restarting-true"` — Rejected: more confusing, same problem.
2. Remove the restarter entirely — Rejected: restart-on-change is a valuable feature.
3. Add a separate annotation for restart — Rejected: couples two orthogonal concerns unnecessarily.

### D2: `CTBRestartEnabled` and `CTBMountEnabled` converge

**Decision:** After this change, when `CTBMountEnabled` returns `true` (i.e., annotation is `"true"`), `CTBRestartEnabled` effectively returns the same. The restarter check simplifies to `CTBMountEnabled`.

**Rationale:**
- The internal API functions can be simplified. `CTBRestartEnabled` can either be removed or made equivalent to `CTBMountEnabled`.
- In the webhook defaulters, the check `CTBRestartEnabled(p.Annotations)` becomes simply `CTBMountEnabled(p.Annotations)` since `"true"` is the only truthy value.

### D3: `"restart-on-change"` becomes an unrecognized value

**Decision:** `"restart-on-change"` is no longer a recognized annotation value. It will be treated as unknown — neither mounting CTB nor triggering restart. Since `"restart-on-change"` was never rolled out in production, no existing workloads are affected.

**Rationale:**
- Unlike a typical breaking change, there are no production pods to migrate. The annotation was experimental and never promoted.
- Anyone who manually set `"restart-on-change"` on a test pod will simply see CTB no longer mounted/restarted — they can update to `"true"` to get the full behavior.

## Risks / Trade-offs

| Risk | Severity | Mitigation |
|------|----------|------------|
| Pods that only wanted CTB mount (no restart) lose restart functionality | Low | No known use case for mount-without-restart exists |

## Migration Plan

No migration is needed — `"restart-on-change"` was never rolled out to production workloads.

## Open Questions

None remaining.
