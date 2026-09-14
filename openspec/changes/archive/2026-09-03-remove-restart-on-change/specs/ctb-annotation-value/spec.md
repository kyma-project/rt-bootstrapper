# CTB Annotation Value Simplification

## ADDED Requirements

### Requirement: Allowed annotation values

The `AnnotationAddClusterTrustBundle` annotation SHALL accept exactly two values:

- `"true"` — Enable CTB volume mounting AND pod restart on hash changes.
- `"false"` — Explicit opt-out: do not mount CTB and do not restart.

Any other value (including the previously supported `"restart-on-change"`) SHALL be treated as unknown and ignored (neither mount nor restart).

#### Scenario: Value is "true"
- **WHEN** a pod has `AnnotationAddClusterTrustBundle` set to `"true"`
- **THEN** the webhook defaulters SHALL mount the ClusterTrustBundle projected volume
- **THEN** the hash SHALL be stamped on the pod (if available from HashHolder)
- **THEN** the CTB restarter SHALL restart the pod when the CTB hash changes

#### Scenario: Value is "false"
- **WHEN** a pod has `AnnotationAddClusterTrustBundle` set to `"false"`
- **THEN** the webhook defaulters SHALL NOT mount the ClusterTrustBundle volume
- **THEN** the CTB restarter SHALL NOT restart the pod

#### Scenario: Value is "restart-on-change" (removed)
- **WHEN** a pod has `AnnotationAddClusterTrustBundle` set to `"restart-on-change"`
- **THEN** the value is treated as unknown and ignored
- **THEN** no CTB volume is mounted
- **THEN** no pod restart is triggered

#### Scenario: Value is unknown
- **WHEN** a pod has `AnnotationAddClusterTrustBundle` set to any unrecognized value
- **THEN** the value is treated as unknown and ignored
- **THEN** no CTB volume is mounted
- **THEN** no pod restart is triggered

### Requirement: CTBMountEnabled behavior

The function `CTBMountEnabled(annotations)` SHALL return `true` only when the annotation value is `"true"`. It SHALL return `false` for `"false"`, `"restart-on-change"`, unknown values, or missing annotations.

#### Scenario: CTBMountEnabled with "true"
- **WHEN** annotations contain `AnnotationAddClusterTrustBundle: "true"`
- **THEN** `CTBMountEnabled` returns `true`

#### Scenario: CTBMountEnabled with "false"
- **WHEN** annotations contain `AnnotationAddClusterTrustBundle: "false"`
- **THEN** `CTBMountEnabled` returns `false`

#### Scenario: CTBMountEnabled with "restart-on-change"
- **WHEN** annotations contain `AnnotationAddClusterTrustBundle: "restart-on-change"`
- **THEN** `CTBMountEnabled` returns `false`

#### Scenario: CTBMountEnabled with missing annotation
- **WHEN** annotations are nil or missing `AnnotationAddClusterTrustBundle`
- **THEN** `CTBMountEnabled` returns `false`

### Requirement: CTBRestartEnabled behavior

The CTB restarter SHALL restart pods only when the annotation value is `"true"`. The restarter SHALL NOT restart pods with `"false"`, `"restart-on-change"`, or any other value.

#### Scenario: Restarter restarts "true" pods with stale hash
- **WHEN** a pod has `AnnotationAddClusterTrustBundle: "true"`
- **AND** the pod's `AnnotationCTBHash` differs from the desired hash
- **THEN** the restarter SHALL delete the pod

#### Scenario: Restarter skips "false" pods
- **WHEN** a pod has `AnnotationAddClusterTrustBundle: "false"`
- **THEN** the restarter SHALL NOT restart the pod regardless of hash

#### Scenario: Restarter skips orphan pods
- **WHEN** a pod has `AnnotationAddClusterTrustBundle: "true"` but no `OwnerReferences`
- **THEN** the restarter SHALL NOT delete the pod
- **THEN** a warning log entry SHALL be recorded
