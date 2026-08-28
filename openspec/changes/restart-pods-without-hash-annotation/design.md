## Context

The Runtime Bootstrapper includes a CTB (ClusterTrustBundle) watcher that monitors changes to a named ClusterTrustBundle resource. When the trust bundle rotates (hash changes), the watcher calls `RestartStalePods` to delete pods whose stamped hash no longer matches the desired hash, triggering their controllers to recreate them with fresh certificates.

The current flow:
1. **Webhook** (`BuildDefaulterAddClusterTrustBundle`): When a pod is created, if CTB is enabled (via pod annotation, namespace annotation, or namespace defaults), the webhook mounts the CTB volume and stamps `ctb-hash` on the pod—but **only if** the pod itself carries `add-cluster-trust-bundle: "true"` as a pod-level annotation AND has owner references.
2. **Restarter** (`RestartStalePods`): Iterates pods, checks `CTBRestartEnabled(pod.Annotations)` which requires `add-cluster-trust-bundle: "true"` on the pod. Pods opted in via namespace-level defaults or namespace annotations don't carry this pod annotation and are skipped.

This creates a gap: pods that received the CTB volume via namespace-level defaults are never restarted when the trust bundle rotates.

## Goals / Non-Goals

**Goals:**
- The webhook SHALL stamp the `ctb-hash` annotation on every CTB-opted-in pod, regardless of opt-in source (pod annotation, namespace annotation, or namespace defaults) and regardless of whether the pod is an orphan.
- Pods with `add-cluster-trust-bundle: "true"` but no `ctb-hash` annotation SHALL be treated as stale and restarted by the restarter.
- Maintain orphan pod protection in the restarter (pods without owner references are still skipped from deletion).

**Non-Goals:**
- Broadening restarter eligibility beyond `CTBRestartEnabled` — the `add-cluster-trust-bundle: "true"` annotation remains the sole restart eligibility signal.
- Modifying the CTB watcher's reconciliation loop or resync interval.
- Adding new annotations or changing the annotation schema.

## Decisions

### Decision 1: Keep `CTBRestartEnabled` as the sole restart eligibility signal

**Choice**: The restarter continues to use `CTBRestartEnabled(pod.Annotations)` (i.e., `add-cluster-trust-bundle: "true"`) as the only eligibility check. Do NOT broaden eligibility to pods that merely carry a `ctb-hash` annotation.

**Rationale**: The `add-cluster-trust-bundle: "true"` annotation is the explicit opt-in for the CTB feature. Pods that carry `ctb-hash` but not the feature annotation may have had it removed intentionally. Keeping a single eligibility signal is simpler and avoids accidental restarts.

**Alternative considered**: Two-pronged eligibility (`CTBRestartEnabled` OR `ctb-hash` present). Rejected because `ctb-hash` alone doesn't indicate current opt-in intent — it could be a leftover from a previous configuration.

**What changes in the restarter**: The eligibility check stays the same; the fix is that pods passing the eligibility check but missing the `ctb-hash` annotation are now correctly treated as stale (empty string ≠ desired hash), which already works via the existing comparison — the bug was that these pods were being skipped before reaching the comparison.

### Decision 2: Treat missing hash as stale

**Choice**: If a pod is restart-eligible (`CTBRestartEnabled` returns true) but has no `ctb-hash` annotation (empty string from map lookup), treat it as stale and delete it.

**Rationale**: A missing hash means the pod was either created before the hash-stamping logic existed, or the webhook couldn't stamp it for some reason. In either case, the pod likely has a stale trust bundle. The comparison `"" == desiredHash` already evaluates to false (since desiredHash is a SHA-256 hex string), so the existing code path already handles this correctly once the pod passes the eligibility check.

### Decision 3: Webhook stamps hash on all CTB-opted-in pods, including orphans

**Choice**: Remove the orphan guard from the webhook's hash-stamping logic. Orphan pods that receive the CTB volume SHALL also receive the `ctb-hash` annotation. The orphan guard only applies in the restarter (deletion), not in the webhook (annotation stamping).

**Rationale**: The hash annotation is informational — it records which trust bundle version the pod was created with. There's no reason to withhold this from orphan pods. The restarter's orphan guard prevents deletion, which is the actual protective behavior. Separating the two concerns (stamping vs. deleting) makes the system clearer.

## Risks / Trade-offs

- **[Risk] Pods opted in via namespace defaults but never stamped with hash are not caught** → Mitigation: The webhook is updated to stamp the hash for all CTB-opted-in pods regardless of opt-in source. This is the root cause fix in `BuildDefaulterAddClusterTrustBundle`.
- **[Risk] Backward compatibility during rollout** → Mitigation: Existing pods with `add-cluster-trust-bundle: "true"` but no `ctb-hash` annotation will be restarted on the first reconciliation after upgrade. This is desired behavior—they get fresh certificates.
- **[Risk] Orphan pods now get ctb-hash stamped** → Mitigation: This is purely informational. The restarter's orphan guard still prevents deletion. Stamping the hash on orphans provides consistency and auditability.
