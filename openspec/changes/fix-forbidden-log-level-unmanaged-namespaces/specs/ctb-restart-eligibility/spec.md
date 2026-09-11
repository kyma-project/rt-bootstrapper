## ADDED Requirements

### Requirement: Forbidden errors on managed namespaces are logged as errors
When `RestartStalePods` encounters a Forbidden error listing pods in a namespace that is a key in `namespaceFeatures`, it SHALL log the event at Error level including the full error detail, and SHALL continue processing remaining namespaces.

#### Scenario: Forbidden error on a managed namespace logs an error
- **WHEN** `RestartStalePods` lists pods in a namespace that is a key in `namespaceFeatures`
- **AND** the API server returns a Forbidden error
- **THEN** the restarter SHALL log a message at Error level containing the namespace name and the error
- **AND** the restarter SHALL skip the namespace and continue with the next one

### Requirement: Forbidden errors on unmanaged namespaces are logged as warnings
When `RestartStalePods` encounters a Forbidden error listing pods in a namespace that is NOT a key in `namespaceFeatures`, it SHALL log the event at Warn level and SHALL continue processing remaining namespaces.

#### Scenario: Forbidden error on an unmanaged namespace logs a warning
- **WHEN** `RestartStalePods` lists pods in a namespace that is NOT a key in `namespaceFeatures`
- **AND** the API server returns a Forbidden error
- **THEN** the restarter SHALL log a message at Warn level containing the namespace name
- **AND** the restarter SHALL skip the namespace and continue with the next one

### Requirement: RestartStalePods accepts a set of managed namespaces
`RestartStalePods` SHALL accept a `managedNamespaces` parameter representing the set of namespace names where RTB is expected to have pod-list RBAC. A nil or empty map SHALL be treated as no namespaces being managed (all Forbidden errors logged as warnings).

#### Scenario: Nil managed namespaces treats all Forbidden errors as warnings
- **WHEN** `RestartStalePods` is called with a nil `managedNamespaces` map
- **AND** a Forbidden error occurs on any namespace
- **THEN** the restarter SHALL log a Warn-level message for that namespace
