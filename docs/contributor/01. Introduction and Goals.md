# Introduction and Goals

## Requirements Overview

Kyma landscapes often run on infrastructure with unique constraints: private container registries that require authentication, custom CA certificate chains, FIPS-mode requirements, or specific registry hostname routing. Kyma modules are not designed to accommodate these differences out of the box.

Runtime Bootstrapper solves this by intercepting every Pod creation in a Kyma runtime cluster and applying the necessary landscape-specific adjustments automatically, before the Pod is scheduled. This removes the requirement for individual Kyma modules to know anything about the infrastructure they run on.

**Core functional requirements:**

| # | Requirement |
|---|---|
| F-1 | Rewrite container image registry hostnames in Pod specs according to a configurable override map. |
| F-2 | Inject an image-pull secret reference into Pod specs so that private registries can be accessed. |
| F-3 | Mount a `ClusterTrustBundle` as a projected volume into every container, enabling TLS communication with BTP backend services. |
| F-4 | Set FIPS-mode environment variables (`KYMA_FIPS_MODE_ENABLED`, `FIPS_MODE_ENABLED`) in every container. |
| F-5 | Synchronize the image-pull secret from the `kyma-system` namespace into every other namespace in the cluster, continuously. |
| F-6 | Apply manipulations selectively based on an opt-in annotation model (namespace-level or Pod-level). |
| F-7 | Read its runtime configuration from a ConfigMap; re-read it on every webhook invocation. |

## Quality Goals

| Priority | Quality Goal | Scenario |
|---|---|---|
| 1 | **Availability of the API server** | The webhook must not block Pod scheduling. If the webhook is unavailable, the Kubernetes API server must continue processing Pod creation requests unimpeded. |
| 2 | **Correctness of manipulation** | Every Pod that opts in (via namespace or Pod annotation, or via the default namespace config) must receive all configured manipulations deterministically and idempotently. |
| 3 | **Operational simplicity** | The webhook must manage its own TLS certificate lifecycle (hot-reload) and automatically update the `caBundle` of its `MutatingWebhookConfiguration`; no manual certificate rotation procedure should be required. |
| 4 | **Security** | HTTP/2 is disabled by default to mitigate stream-cancellation CVEs. TLS 1.3 is the minimum version enforced by the webhook server. The webhook runs with a high-priority class to avoid being displaced by user workloads. |

## Stakeholders

| Role | Expectations |
|---|---|
| Kyma Infrastructure Manager (KIM) | Installs and configures Runtime Bootstrapper on every provisioned Kyma runtime. Manages the lifecycle of the shared resources (pull secret, ConfigMap, `ClusterTrustBundle`) in `kyma-system`. |
| Kyma Lifecycle Manager (KLM) | Deploys Kyma modules via high-level resources (Deployments, DaemonSets). Pods created from those resources are intercepted and adjusted by Runtime Bootstrapper without KLM involvement. |
| Kyma module teams | Their modules run unmodified inside landscapes with private registries and custom CA chains; Runtime Bootstrapper makes that possible transparently. |
| Kyma runtime operators / customers | Can opt their own namespaces or Pods into the manipulation mechanism by adding the appropriate annotations. |
| Runtime Bootstrapper contributors | Need to understand the architecture to extend existing manipulations or add new ones. |
