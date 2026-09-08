## ADDED Requirements

### Requirement: Webhook stamps ctb-hash on all CTB-opted-in pods
The webhook's `BuildDefaulterAddClusterTrustBundle` SHALL stamp the `ctb-hash` annotation on every pod that receives the CTB volume, regardless of whether the opt-in came from a pod-level annotation, namespace annotation, or namespace default features. The hash stamp SHALL NOT be limited to pods where `CTBRestartEnabled(pod.Annotations)` returns true.

#### Scenario: Pod opted in via namespace defaults receives hash stamp
- **WHEN** a pod is created in a namespace that has CTB enabled via `namespaceFeatures` config defaults
- **AND** the pod does not carry `add-cluster-trust-bundle: "true"` as a pod-level annotation
- **AND** the pod has owner references
- **AND** the hash holder has a non-empty hash
- **THEN** the webhook SHALL mount the CTB volume AND stamp the `ctb-hash` annotation on the pod

#### Scenario: Pod opted in via namespace annotation receives hash stamp
- **WHEN** a pod is created in a namespace with `add-cluster-trust-bundle: "true"` as a namespace annotation
- **AND** the pod does not carry the annotation itself
- **AND** the pod has owner references
- **AND** the hash holder has a non-empty hash
- **THEN** the webhook SHALL mount the CTB volume AND stamp the `ctb-hash` annotation on the pod

#### Scenario: Orphan pod receives hash stamp
- **WHEN** a pod is created that is opted in for CTB (by any mechanism)
- **AND** the pod has no owner references
- **THEN** the webhook SHALL stamp the `ctb-hash` annotation

### Requirement: Restarter considers pods with ctb-hash annotation as restart-eligible
The `RestartStalePods` function SHALL treat any pod carrying the `ctb-hash` annotation as eligible for restart, regardless of whether the pod also has the `add-cluster-trust-bundle` annotation.

#### Scenario: Pod with ctb-hash but without add-cluster-trust-bundle annotation is not restarted
- **WHEN** a pod has the `ctb-hash` annotation with a value different from the desired hash
- **AND** the pod does NOT have `add-cluster-trust-bundle: "true"` annotation
- **AND** the pod has owner references
- **THEN** `RestartStalePods` SHALL NOT delete the pod

#### Scenario: Pod without ctb-hash but with add-cluster-trust-bundle annotation is restarted
- **WHEN** the pod has `add-cluster-trust-bundle: "true"` annotation
- **AND** the pod does not have the `ctb-hash` annotation 
- **AND** the pod has owner references
- **THEN** `RestartStalePods` SHALL delete the pod

#### Scenario: Pod with matching ctb-hash is not restarted
- **WHEN** a pod has the `ctb-hash` annotation with a value equal to the desired hash
- **THEN** `RestartStalePods` SHALL NOT delete the pod

### Requirement: Pods without ctb-hash annotation but with CTB restart enabled are treated as stale
When a pod has `add-cluster-trust-bundle: "true"` but no `ctb-hash` annotation (empty string), the pod SHALL be treated as stale and eligible for restart.

#### Scenario: Pod with CTB annotation but missing hash is restarted
- **WHEN** a pod has `add-cluster-trust-bundle: "true"` annotation
- **AND** the pod does NOT have the `ctb-hash` annotation
- **AND** the pod has owner references
- **THEN** `RestartStalePods` SHALL delete the pod

### Requirement: Orphan pod protection remains unchanged
Pods without owner references SHALL NOT be deleted regardless of their annotation state.

#### Scenario: Orphan pod with CTB annotation is not deleted
- **WHEN** the pod has no owner references
- **AND** the pod has `add-cluster-trust-bundle: "true"` annotation
- **THEN** `RestartStalePods` SHALL NOT delete the pod and SHALL log a warning
