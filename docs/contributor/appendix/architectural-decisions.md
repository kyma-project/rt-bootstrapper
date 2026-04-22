# Architectural Decisions

## Technical Design

Several architectural decisions were made during the Kyma architecture meeting and the implementation phase. These decisions were primarily driven by technical constraints and the need for timely solutions.

### High-Level Design

![High Level Architecture](./assets/high-level-arch.drawio.svg)

**Components:**
 
 1. **KIM (Kyma Infrastructure Manager):** Deploys the webhook and shared resources to Kyma runtimes.
 2. **API Server:** The Kubernetes API server calls the manipulation webhook to intercept the Pod manifest before it gets applied.
 3. **RT Bootstrapper:** Modifies Pod manifests and applies landscape-specific adjustments (e.g., adding pull-secret or rewriting image-registry host-names, etc.).
 4. **Workload:** The manipulated workload is adjusted to the landscape-specific setup.
 5. (Optional)  The workload can use shared resources (e.g., pull-secrets, cluster-trust-bundles, etc.).
 

## Technical Requirements

### Manipulation is Limited to Pods
The webhook only manipulates Pod resources. Other resources, such as StatefulSets, DaemonSets, and Deployments, are ignored. This is required to avoid conflicts between Kyma Lifecycle Manager (KLM) and Kyma Infrastructure Manager (KIM). KLM regularly processes the resources it deployed (for example, Deployments of operators). If the webhook were to modify these deployments, the KLM would revert the modifications regularly, and both processes would "fight" against each other. To avoid such a situation, we agreed that KLM will never deploy Pods, but high-level resources like Deployments, DaemonSets, StatefulSets, etc. The drawback of this decision is that the deployed Pod can include different values compared to its definition within a Deployment, StatefulSet, DaemonSet, etc., which may be confusing for engineers or developers reviewing a Pod definition in Kubernetes who are unaware of the webhook's existence and its adjustments.

### Non-Blocking Webhook
The admission webhook must be configured as a non-blocking processing step for API-server requests. This means that the API server continues processing the request when the webhook cannot be invoked. This decision ensures that the API server continues to process requests even when the webhook is temporarily unavailable. The decision introduces the risk that Pods get scheduled without being manipulated.

### Detection of Non-Manipulated Resources Is Not Part of the Webhook
We agreed that the webhook is exclusively responsible for manipulating the manifest of Pods during their creation phase. If a Pod gets scheduled without being processed by the webhook (for example, when the webhook is temporarily down), the Pod might miss critical adjustments and, in the worst case, may not start up properly. To address this issue, a housekeeping process implemented outside of the webhook regularly scans all Pods for any missing manipulations. If such Pods are identified, the housekeeping process restarts them (during the re-creation, the webhook is invoked, and the manipulations are applied).

### Opt-In Approach
We agreed that Pods are processed by the webhook only if one of the following conditions is fulfilled:

1. The configuration of the Webhook defines a list of mandatory manipulations for the namespace. This ensures that any Pod in Kyma-managed namespaces is processed.
2. The namespace is annotated to receive particular manipulations.
3. The Pod itself is annotated to receive manipulations.

This also enables customers to opt into this modification mechanism by annotating either their own namespace or the Pod manifests accordingly.

### Webhook Configuration
The webhook retrieves a default configuration that specifies the number of manipulations to apply to all Pods in particular namespaces. Customers or other workloads cannot modify this configuration. 

By default, the configuration considers only Kyma-managed namespaces (e.g., `kyma-system`, `istio-system`, etc.) to avoid conflicts with customer-owned namespaces.

### Applied Manipulations
The webhook supports multiple manipulations. The default configuration, managed by KIM, determines which manipulation is used.

### Resource Synchronization
To adjust the workloads to landscape-specific setups, several resources must be published in the Kyma runtime:

1. Pull secrets to authenticate at private container registries.
2. `ClusterTrustBundle` used to store certificate chains (needed for secured backend communication).
3. The configuration of the Webhook itself.

The Kyma backend ensures that such resources are synchronized from Kyma Control Plane (KCP) to the Kyma runtime `kyma-system` namespace. For more information on this mechanism, see [Runtime Configuration Synchronization Using Controller Loop](./resource-synchronisation.md).

Some resources are namespace-scoped and must be replicated to all other namespaces in the cluster (e.g., pull secrets). The Runtime Bootstrapper webhook includes a dedicated controller that synchronizes such resources into all Kyma runtime namespaces.
