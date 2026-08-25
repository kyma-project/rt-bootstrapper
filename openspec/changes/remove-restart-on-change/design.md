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

### D3: Upward-compatible migration — nothing blocks, behavior just changes

**Decision:** Existing pods with `"restart-on-change"` will continue to work. On next reconciliation:
1. The defaulter sees `"restart-on-change"` is no longer recognized as a valid truthy value.
2. The annotation is effectively treated as unknown — but **expanded annotations** may normalize it.
3. Hash stamping triggers, and the restarter handles restart normally.

**Important:** Since `"restart-on-change"` is no longer a recognized value, it will NOT match `CTBMountEnabled` after the code change, meaning existing pods with `"restart-on-change"` will NOT be re-defaulted unless their annotation is `"true"`. **This is the breaking change.** Users must update existing pod annotations to `"true"` to maintain the current behavior.

## Risks / Trade-offs

| Risk | Severity | Mitigation |
|------|----------|------------|
| Pods that only wanted CTB mount (no restart) lose restart functionality | Low | No known use case for mount-without-restart exists |

## Migration Plan

1. **Code change** (this PR/merge): Remove `CTBValueRestartOnChange`, merge semantics into `"true"`.
2. **Docs update**: Update any documentation referencing `"restart-on-change"` to use `"true"` instead.
3. **User communication**: If any users have pods using `"restart-on-change"`, they need to update to `"true"`.

## Open Questions

None remaining.
