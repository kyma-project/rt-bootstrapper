### Requirement: Restart eligibility includes pods with ctb-hash annotation
The `RestartStalePods` function SHALL treat any pod carrying the `rt-bootstrapper.kyma-project.io/ctb-hash` annotation as eligible for restart, regardless of whether the pod also has the `rt-cfg.kyma-project.io/add-cluster-trust-bundle` annotation on the pod itself.

#### Scenario: Pod opted-in via namespace annotation is restarted on CTB change
- **WHEN** a namespace has `rt-cfg.kyma-project.io/add-cluster-trust-bundle: "true"` annotation
- **AND** a pod in that namespace was mutated by the webhook (has `ctb-hash` annotation)
- **AND** the pod does NOT have `rt-cfg.kyma-project.io/add-cluster-trust-bundle: "true"` as a pod-level annotation
- **AND** the pod has owner references
- **AND** the ClusterTrustBundle changes (new hash differs from the pod's `ctb-hash`)
- **THEN** `RestartStalePods` SHALL delete the pod

#### Scenario: Pod opted-in via namespace default features is restarted on CTB change
- **WHEN** a namespace is configured in `namespaceFeatures` with CTB enabled
- **AND** a pod in that namespace was mutated by the webhook (has `ctb-hash` annotation)
- **AND** the pod does NOT have `rt-cfg.kyma-project.io/add-cluster-trust-bundle: "true"` as a pod-level annotation
- **AND** the pod has owner references
- **AND** the ClusterTrustBundle changes
- **THEN** `RestartStalePods` SHALL delete the pod

#### Scenario: Pod opted-in via pod-level annotation is still restarted
- **WHEN** a pod has `rt-cfg.kyma-project.io/add-cluster-trust-bundle: "true"` as a pod-level annotation
- **AND** the pod has `ctb-hash` annotation
- **AND** the pod has owner references
- **AND** the ClusterTrustBundle changes
- **THEN** `RestartStalePods` SHALL delete the pod

#### Scenario: Pod with matching ctb-hash is not restarted
- **WHEN** a pod has the `ctb-hash` annotation with a value equal to the desired hash
- **THEN** `RestartStalePods` SHALL NOT delete the pod

### Requirement: Pods without ctb-hash and without pod-level CTB annotation are not restarted
Pods that do not carry either the `ctb-hash` annotation or the `add-cluster-trust-bundle: "true"` pod-level annotation SHALL NOT be considered for restart.

#### Scenario: Regular pod without CTB annotations is not restarted
- **WHEN** a pod has no `ctb-hash` annotation
- **AND** the pod has no `rt-cfg.kyma-project.io/add-cluster-trust-bundle` annotation
- **THEN** `RestartStalePods` SHALL NOT delete the pod

### Requirement: Pods with add-cluster-trust-bundle but no ctb-hash are treated as stale
Pods that have `add-cluster-trust-bundle: "true"` as a pod-level annotation but no `ctb-hash` annotation (e.g., created before the hash-stamping logic) SHALL be treated as stale and restarted.

#### Scenario: Pod with CTB annotation but missing hash is restarted
- **WHEN** a pod has `rt-cfg.kyma-project.io/add-cluster-trust-bundle: "true"` annotation
- **AND** the pod does NOT have the `ctb-hash` annotation
- **AND** the pod has owner references
- **THEN** `RestartStalePods` SHALL delete the pod (empty hash ≠ desired hash)

### Requirement: Orphan pod protection remains unchanged
Pods without owner references SHALL NOT be deleted by the restarter regardless of their annotation state.

#### Scenario: Orphan pod with ctb-hash is not deleted
- **WHEN** a pod has `ctb-hash` annotation with a stale hash
- **AND** the pod has no owner references
- **THEN** `RestartStalePods` SHALL NOT delete the pod and SHALL log a warning
