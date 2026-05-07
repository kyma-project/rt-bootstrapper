# Architecture Constraints

## Technical Constraints

| Constraint | Rationale |
|---|---|
| **Only Pods are manipulated, never higher-level resources** | KLM regularly reconciles the Deployments, DaemonSets, and StatefulSets it owns. Mutating those resources would cause a perpetual conflict between KLM reverting changes and the webhook re-applying them. Because KLM never creates bare Pods directly, the webhook safely intercepts only at the Pod level. |
| **Webhook failure must be non-blocking (`failurePolicy: Ignore`)** | The webhook must not become a single point of failure for the entire Kubernetes API server. When the webhook is temporarily unavailable, Pod creation requests proceed unmodified. The trade-off (unmanipulated Pods) is handled by an external housekeeping process. |
| **No CRD – configuration via ConfigMap** | Runtime Bootstrapper has no Custom Resource Definition. Its configuration (`rt-bootstrapper-config.json`) is stored in a ConfigMap managed by KIM. This removes the bootstrapping problem of needing a CRD to be present before the component that serves it starts. |
| **Go 1.26+ with FIPS-mode build flag** | All `go` invocations use `GOFIPS140=v1.0.0` to produce FIPS-140-compliant binaries. This is a hard requirement for SAP BTP-regulated landscapes. |
| **TLS 1.3 minimum, HTTP/2 disabled by default** | HTTP/2 is disabled (unless `--enable-http2` is passed) to mitigate CVE-2023-44487 (Rapid Reset) and CVE-2023-39325 (Stream Cancellation). The webhook server enforces TLS 1.3 as the minimum version. |
| **Deployed in `kyma-system` namespace** | All Kubernetes resources belonging to Runtime Bootstrapper (Deployment, Service, RBAC, etc.) are placed in the `kyma-system` namespace. |
| **High scheduling priority** | The manager runs with a `PriorityClass` value of `2 100 000` to prevent user workloads from starving the webhook. |

## Organizational Constraints

| Constraint | Rationale |
|---|---|
| **Installed and configured exclusively by KIM** | In production, Runtime Bootstrapper is not installed manually. KIM provisions it as part of the Kyma runtime lifecycle. Self-managed or direct `kubectl apply` installations are unsupported in SAP BTP Kyma runtime. |
| **External housekeeping for non-manipulated Pods** | Detecting and remediating Pods that were created while the webhook was down is out of scope for Runtime Bootstrapper itself. Currently no process handles that and manual intervention is needed. |
| **Configuration is read-only from the workload perspective** | Namespaces and Pods can opt in or out via annotations, but the default configuration (which namespaces are always processed, which features are available) is exclusively managed by KIM via the ConfigMap. |
