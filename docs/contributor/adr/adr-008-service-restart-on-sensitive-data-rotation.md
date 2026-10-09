# ADR-008 – Service Restart on Sensitive Data Rotation

**Status:** Proposed

## Context

RT Bootstrapper, in combination with Kyma Infrastructure Manager (KIM), distributes sensitive data and certificates — such as `ClusterTrustBundle` — to all Kyma runtimes. Depending on the implementation of the consuming application, a change to these values may require a service restart to take effect (for example, when a certificate is loaded only once at startup and not reloaded from the filesystem or Kubernetes API at runtime).

SAP security requirements mandate that sensitive data, including certificates, must be rotated periodically. This means rotation is not an exceptional event but a regular operational procedure that the system must handle reliably.

The question is: **who or what triggers the restart of affected services when sensitive data changes?**

Two principal approaches exist:

### Option A – Central Restart Mechanism

A dedicated component (or operational script) watches for changes to sensitive data and restarts affected workloads automatically. This could be implemented by:

- Adopting and extending an existing open-source tool such as [Stakater/Reloader](https://github.com/stakater/Reloader), which currently supports watching Secrets and ConfigMaps but not CRDs (such as `ClusterTrustBundle`). Extending it to support CRDs would be required.
- Running an operator/controller developed in-house that restarts Pods or rolls Deployments when the relevant resources change.
- Having SREs execute a manual restart script after confirming that the new value has been propagated to all target SKRs.

### Option B – Application-Owned Restart / Hot Reload

Each consuming application is responsible for detecting that its sensitive data has changed and reloading or restarting itself accordingly. This can be done by watching the Kubernetes API for changes to the relevant resource, periodically polling, or implementing inotify-based file watches if the data is mounted as a volume.

## Decision

**To be decided.**

## Considered Options and Trade-offs

### Option A – Central Restart Mechanism

**Pros:**

- Consuming applications require no changes to support rotation; the mechanism works generically across all services.
- Provides a single, auditable place to observe and control restart behaviour during rotation events.
- The SRE-script variant is low-cost to implement and avoids deploying additional software to each SKR.

**Cons:**

- A central component that rolls Deployments or deletes Pods across namespaces requires broad RBAC permissions (for example, `patch Deployments` or `delete Pods` cluster-wide), which increases the attack surface and contradicts the principle of least privilege.
- RT Bootstrapper is not the natural owner of this responsibility: some sensitive data (e.g. `ClusterTrustBundle`) is published by KIM, not RT Bootstrapper. RT Bootstrapper does not have visibility into when KIM has finished propagating a new value to a specific SKR, so triggering a restart at the wrong moment could leave a service restarted before the new data is available.
- Services are restarted independently of their internal state (for example, mid-transaction, during a reconciliation loop, or while serving a request burst). This can cause service interruptions that are unrelated to any application bug.
- If a restart causes an application to fail to start (for example, because the new certificate is malformed or the trust chain is incomplete), the team owning the restart mechanism becomes the incident owner, even though the root cause lies elsewhere. This creates an unfavourable risk transfer.
- The SRE-script variant introduces a manual, error-prone step and does not scale as the number of SKRs grows. It also requires coordination to ensure the new value has fully propagated before the restart is triggered.
- Extending Stakater/Reloader to watch CRDs adds maintenance burden and creates a dependency on a third-party codebase. Any upstream breaking changes must be tracked and absorbed.

### Option B – Application-Owned Restart / Hot Reload

**Pros:**

- Each application restarts (or reloads) only when it is in a safe state to do so, reducing the risk of service interruptions caused by externally triggered restarts at inconvenient times.
- Fulfils SAP rotation requirements by design: applications that handle rotation internally are inherently capable of meeting the requirement without depending on an external component.
- Eliminates the need for a broad-permission central component, reducing the cluster-wide RBAC footprint.
- Responsibility for restart behaviour stays with the team that understands the application's lifecycle and internal state best.
- Aligns with the Kubernetes ecosystem convention of building resilient, self-managing applications (for example, certificate manager's `csi-driver` sidecars that reload certificates transparently).

**Cons:**

- Requires all consuming applications to be modified or verified to support hot reload or self-restart on data changes. This is a non-trivial effort across multiple teams.
- Applications that cannot be modified (for example, third-party or off-the-shelf components) cannot adopt this approach without a wrapper or sidecar.
- Inconsistent implementation quality across teams may result in some applications reloading correctly while others silently continue to use stale data.
- Increases per-application complexity and requires teams to understand and test rotation scenarios individually.

## Consequences (dependent on decision)

If **Option A** is chosen:

- A central restart component must be designed, deployed, and operated on every SKR.
- RBAC permissions for the component must be scoped as narrowly as possible and reviewed from a security perspective.
- The timing of restarts must be coordinated with KIM's propagation cycle to avoid restarting services before the new data is available.
- A clear ownership model for restart-induced incidents must be established before the mechanism goes live.
- If Stakater/Reloader is reused, the upstream project must be forked or extended to support CRD watches, and a maintenance strategy must be defined.

If **Option B** is chosen:

- All applications that consume sensitive data distributed by RT Bootstrapper or KIM must be audited and, where necessary, updated to implement reload or safe self-restart logic.
- A conformance requirement or checklist should be introduced to verify that new services handle rotation correctly before they are deployed to SKRs.
- Third-party or unmodifiable applications require an alternative solution (for example, a per-Pod sidecar or a scoped restart controller limited to those specific workloads).
