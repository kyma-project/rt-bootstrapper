
# Runtime Configuration Synchronization Using Controller Loop

Runtime Boostrapper synchronises several resources between Kyma Control Plane (KCP) and Kyma runtimes. Some webhook features require specific resources to work (for example, a pull secret to access a private container registry, `ClusterTrustBundle` to interact with BTP backend services, etc.).

The following described behavior represents an interim solution. A long‑term architecture is planned in which the controller loop will directly synchronize configuration data to the runtimes without relying on indirect signaling.


## Components

![Runtime Bootstrapper Architecture](./assets/old-arch-rt-bootstrapper.drawio.svg)

### Controller Loop

A controller loop is a custom Kubernetes controller that observes changes to selected cluster resources in KCP and initiates downstream actions. The watched resources are copied to  Kyma runtimes. Each resource change must be synchronized from KCP to Kyma runtimes.

### Watched Resources

The controller loop monitors the following Kubernetes objects:

| Resource Type | Purpose | 
| --- | --- | 
| Pull secret | Provides authentication credentials for pulling container images from private registries. | 
| `ClusterTrustBundle` | Supplies trust anchors (for example, CA certificates) required by runtimes that interact with the BTP backend services. | 
| Webhook ConfigMap | Contains configuration for the Runtime Bootstrapper webhook. | 


### Runtime Custom Resource

A custom resource (CR) representing a managed runtime instance.

Kyma Infrastructure Manager (KIM) reacts to Runtime CR labels to determine if a runtime requires reconciliation.

### Kyma Infrastructure Manager (KIM)

The Infrastructure Manager watches the Runtime CR for modifications. If it detects the label to force a reconciliation, it reconciles the target SKR and also synchronizes the share resources.


## Current Behavior (Interim Solution)

### Change Detection

The controller loop continuously watches the pull secret, `ClusterTrustBundle`, and the webhook ConfigMap.

Whenever one of these resources is created, updated, or deleted, the controller receives an event.


### Triggering Reconciliation Through Labeling

Upon detecting a change, the controller loop performs the following steps:

1. Identifies all affected `Runtime` CR objects.
2. Applies or updates a specific label (for example, [`operator.kyma-project.io/force-patch-reconciliation=true`](https://github.com/kyma-project/kyma-infrastructure-manager/blob/c1d2f48a9b446b3374528278b46ea9be23ff622a/pkg/reconciler/annotations_utils.go#L4C32-L4C83)) on each `Runtime` CR.
3. KIM observes this label change.
4. KIM reconciles the corresponding runtimes to ensure they receive the updated configuration.

This mechanism uses the `Runtime` CR label as a signaling channel between the controller loop and the Kyma Infrastructure Manager.


### Rationale for the Interim Approach

The labeling strategy provides a lightweight and low‑risk integration path with the following advantages:

* No direct modification of runtime resources is required.
* Existing reconciliation logic in KIM remains unchanged.
* The controller loop only signals intent rather than performing the full synchronization.

This allows incremental rollout and testing of the controller loop without impacting runtime stability.


## Long‑Term Architecture

In the future design, the controller loop will no longer rely on labeling `Runtime` CR objects to trigger reconciliation. Instead, it will directly propagate configuration changes to the runtimes without KIM involvement.

![Runtime Bootstrapper Architecture](./assets/new-arch-rt-bootstrapper.drawio.svg)

### Planned Behavior

When a watched resource changes, the following actions will take place:

1. The controller loop will use a dedicated CR to manage the lifecycle of Runtime Bootstrapper:
    1. A newly created CR will install Runtime Boostrapper on a Kyma runtime.
    2. The status applied to Runtime Bootstrapper will be reflected in the CR status.
    3. Deletion of the CR will trigger the undeployment of Runtime Bootstrapper from the Kyma runtime.
2. The controller loop retrieves the updated value (for example, new pull secret, updated trust bundle, modified webhook configuration).
3. The controller loop writes the updated data directly into the runtime’s configuration or associated Kubernetes objects.
4. KIM no longer needs to detect labels or perform reconciliation for these specific updates.

### Benefits of the Long‑Term Approach

* Reduced latency: Changes are applied immediately without waiting for external reconciliation loops.
* Lower complexity: Reduces complexity in KIM and removes the indirection of using labels as signals.
* Improved consistency: Ensures runtimes always reflect the latest cluster configuration.
* Clearer separation of responsibilities:
  * Controller loop handles configuration propagation.
  * The controller loop handles configuration propagation.
  * The controller manages the lifecycle of Runtime Boostrapper
  * KIM focuses on lifecycle management of Kyma runtimes.
  * KIM is unaware of Runtime Bootstrapper.



## Migration-Related Considerations

Transitioning from the interim to the long‑term solution requires the following:

* Clear separation and decoupling between KIM and the Runtime Bootstrapper components.
* Existing code that resides in KIM can be refactored and moved to Runtime Bootstrapper, which simplifies the KIM codebase and reduces the number of reconciliation steps.
* Updating the controller loop to implement direct synchronization logic.
* Ensuring KIM no longer depends on RuntimeCR label changes for these configuration types.
* Introducing versioning or compatibility checks to avoid partial updates.
* Validating that runtimes can safely accept live updates to trust bundles, pull secrets, or webhook configuration.



## Summary

The current system uses a Kubernetes controller loop to detect changes in key configuration resources and signals KIM by labeling `Runtime` CR objects. This approach serves as a temporary mechanism to ensure runtimes are reconciled when configuration changes occur.

The long‑term solution will perform the following actions:

1. Deploy the Runtime Bootstrapper webhook on Kyma runtimes
2. Use its own CR to manage the Runtime Boostrapper lifecycle and status
3. Eliminate the signaling step to KIM and instead allow the controller loop to directly propagate configuration updates to the runtimes, improving efficiency, reducing complexity, and strengthening consistency across the system
