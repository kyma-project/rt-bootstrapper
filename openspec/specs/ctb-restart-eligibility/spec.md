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

### Requirement: Forbidden errors on managed namespaces are logged as errors
When `RestartStalePods` encounters a Forbidden error listing pods in a namespace that is a key in `namespaceFeatures`, it SHALL log the event at Error level including the full error detail, and SHALL continue processing remaining namespaces.

#### Scenario: Forbidden error on a managed namespace logs an error
- **WHEN** `RestartStalePods` lists pods in a namespace that is a key in `namespaceFeatures`
- **AND** the API server returns a Forbidden error
- **THEN** the restarter SHALL log a message at Error level containing the namespace name and the error
- **AND** the restarter SHALL skip the namespace and continue with the next one

### Requirement: Only managed namespaces are visited by RestartStalePods
`RestartStalePods` SHALL only attempt to list pods in namespaces that are keys in `managedNamespaces`. Namespaces not in this set SHALL be skipped entirely, without issuing any API call or log entry for them.

This is required because the pod informer cache is restricted to the same set of namespaces. Calling `c.List` for a namespace outside the cache returns `"unknown namespace for the cache"`, which is not a Forbidden error and would propagate as a reconciler error.

#### Scenario: Unmanaged namespace is silently skipped
- **WHEN** `RestartStalePods` iterates cluster namespaces
- **AND** a namespace is NOT a key in `managedNamespaces`
- **THEN** the restarter SHALL skip that namespace without issuing a pod List call
- **AND** SHALL NOT emit any log entry for that namespace

#### Scenario: Nil managed namespaces skips all namespaces
- **WHEN** `RestartStalePods` is called with a nil `managedNamespaces` map
- **THEN** every namespace is treated as unmanaged and skipped
- **AND** no pods are deleted

### Requirement: Pod informer cache is restricted to managed namespaces
The controller-runtime manager SHALL configure the pod informer cache to watch only the namespaces listed as keys in `namespaceFeatures`. This prevents the cache from attempting a cluster-scoped pod LIST/WATCH, which the RTB service account is not permitted to perform and which would cause a `controller-runtime.cache.UnhandledError` retry loop.

#### Scenario: Cache does not attempt cluster-scoped pod watch
- **WHEN** the manager starts
- **AND** `namespaceFeatures` contains namespaces e.g. `kyma-system`, `istio-system`, `sap-transp-proxy-system`
- **THEN** the pod informer SHALL only establish watches for those namespaces
- **AND** SHALL NOT attempt a cluster-scoped `LIST /api/v1/pods`

#### Scenario: Cache namespace list is derived from config at startup
- **WHEN** the manager is initialised
- **THEN** it SHALL read `namespaceFeatures` from the config before calling `ctrl.NewManager`
- **AND** pass the namespace set to `cache.Options.ByObject` for `*v1.Pod`

### Requirement: RestartStalePods accepts a set of managed namespaces
`RestartStalePods` SHALL accept a `managedNamespaces map[string]bool` parameter representing the set of namespace names where RTB is expected to have pod-list RBAC. Only namespaces present in this map are visited; a nil map causes all namespaces to be skipped.
