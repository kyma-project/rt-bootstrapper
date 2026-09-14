## Context

The Runtime Bootstrapper's CTB (ClusterTrustBundle) feature has three opt-in mechanisms for pods:
1. **Pod-level annotation**: `rt-cfg.kyma-project.io/add-cluster-trust-bundle: "true"` on the pod itself
2. **Namespace annotation**: The same annotation on the namespace
3. **Namespace default features**: Configured via `namespaceFeatures` in the bootstrapper config

The mutating webhook (`BuildDefaulterAddClusterTrustBundle`) correctly handles all three sources: it mounts the CTB projected volume and stamps `rt-bootstrapper.kyma-project.io/ctb-hash` on every opted-in pod regardless of opt-in source.

The CTB watcher (`CTBWatcher`) monitors the named ClusterTrustBundle for changes and calls `RestartStalePods` to delete pods with stale hashes. However, `RestartStalePods` uses `CTBRestartEnabled(pod.Annotations)` as the eligibility check, which looks for `rt-cfg.kyma-project.io/add-cluster-trust-bundle: "true"` on the **pod's own annotations**. The webhook never copies this annotation onto the pod — it only stamps `ctb-hash`. Therefore, pods opted-in via namespace-level mechanisms are silently skipped by the restarter and keep stale CA certificates.

## Goals / Non-Goals

**Goals:**
- Pods opted-in for CTB via namespace annotation or namespace default features SHALL be restarted when the ClusterTrustBundle changes, just like pods opted-in via pod-level annotation.
- The `ctb-hash` annotation (already stamped by the webhook on all CTB-opted-in pods) SHALL be the primary signal for restart eligibility.
- Orphan pod protection (pods without ownerReferences are not deleted) SHALL remain unchanged.

**Non-Goals:**
- Modifying the webhook's pod mutation logic — it already correctly stamps `ctb-hash` on all opted-in pods.
- Changing the CTB watcher's reconciliation loop or resync interval.
- Adding new annotations or changing the annotation schema.
- Handling pods that have `add-cluster-trust-bundle: "true"` as a pod annotation but no `ctb-hash` (this is already handled: missing hash ≠ desired hash → pod is deleted).

## Decisions

### Decision 1: Use `ctb-hash` annotation presence as the restart eligibility signal

**Choice**: Replace `CTBRestartEnabled(pod.Annotations)` in `RestartStalePods` with a check for the presence of the `ctb-hash` annotation. A pod is restart-eligible if and only if it carries `rt-bootstrapper.kyma-project.io/ctb-hash`.

**Rationale**: The `ctb-hash` annotation is already stamped by the webhook on every CTB-opted-in pod, regardless of opt-in source. It is the most reliable and consistent signal that a pod participates in the CTB feature. Unlike `add-cluster-trust-bundle`, which is a configuration annotation that may only exist on the namespace, `ctb-hash` is a runtime annotation placed directly on the pod by the webhook.

**Alternative considered**: Copy the `add-cluster-trust-bundle: "true"` annotation from namespace to pod in the webhook. Rejected because:
- It conflates configuration intent (namespace-level) with runtime state (pod-level).
- The `ctb-hash` annotation already serves as the pod-level marker and is more precise.
- Adding another annotation increases the surface area and creates ambiguity about which annotation is authoritative.

**Alternative considered**: Check namespace annotations in the restarter (look up the pod's namespace to see if CTB is enabled). Rejected because:
- It adds API calls per pod (list namespaces, check annotations) increasing reconciliation latency.
- It creates a dependency on namespace state that could be inconsistent with pod state.
- The `ctb-hash` annotation is already available on the pod and is self-contained.

### Decision 2: Introduce `CTBHashPresent` helper and update `CTBRestartEnabled`

**Choice**: Add a new helper `CTBHashPresent(annotations) bool` that returns true when the `ctb-hash` annotation key exists. Update `CTBRestartEnabled` to return true when either the `add-cluster-trust-bundle: "true"` annotation OR the `ctb-hash` annotation is present.

**Rationale**: Updating `CTBRestartEnabled` (which is the function used by the restarter) ensures backward compatibility — existing call sites don't need changes. Adding the `CTBHashPresent` helper makes the individual check reusable and testable. Pods with only the pod-level `add-cluster-trust-bundle: "true"` annotation (but missing `ctb-hash` due to pre-upgrade creation) remain eligible for restart.

### Decision 3: Keep orphan protection in the restarter unchanged

**Choice**: The orphan guard (`len(pod.OwnerReferences) == 0 → skip`) remains after the eligibility check. No changes to orphan handling.

**Rationale**: Orphan protection is orthogonal to eligibility. The restarter should identify all stale pods but only delete those with owners who will recreate them.

## Risks / Trade-offs

- **[Risk] Pods that had CTB removed could still carry a stale `ctb-hash` annotation** → Mitigation: If CTB is disabled for a namespace, new pods won't get `ctb-hash` stamped. Existing pods with stale `ctb-hash` will be restarted once (their hash won't match), and the new pods created without CTB won't carry the annotation. This is a one-time self-healing behavior.
- **[Risk] Backward compatibility during rollout** → Mitigation: After upgrade, the first CTB change will correctly restart all pods with `ctb-hash` (including those previously skipped). Pods without `ctb-hash` but with `add-cluster-trust-bundle: "true"` are also handled by the updated `CTBRestartEnabled` logic.
- **[Trade-off] Slightly broader restart surface** → By using `ctb-hash` presence, any pod that was ever mutated by the CTB webhook becomes restart-eligible. This is the correct behavior — if the webhook mounted the CTB volume, the pod needs fresh certificates when the bundle rotates.
