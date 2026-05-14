# Glossary

| Term | Definition |
|---|---|
| Admission webhook | A Kubernetes extension point that intercepts API server requests (create, update, delete) before they are persisted. A _mutating_ admission webhook can modify the resource. See also: `MutatingWebhookConfiguration`. |
| `availableFeatures` | The allowlist of feature annotation keys that Runtime Bootstrapper is permitted to activate. Configured in the ConfigMap. Only defaulters whose key appears in this list are registered at startup. |
| `caBundle` | The base64-encoded CA certificate embedded in the `MutatingWebhookConfiguration` so the Kubernetes API server can verify the webhook's TLS certificate. Automatically kept in sync by Runtime Bootstrapper. |
| certwatcher | A `controller-runtime` helper that watches TLS certificate files on disk using `fsnotify` and reloads them without restarting the process. |
| ClusterTrustBundle | A cluster-scoped Kubernetes resource (alpha/beta feature gate) that stores a bundle of trusted CA certificates. Runtime Bootstrapper can mount it as a projected volume into Pods to enable TLS communication with BTP backend services. |
| ConfigMap `rt-bootstrapper-config` | The Kubernetes ConfigMap in `kyma-system` that holds the JSON configuration document (`rt-bootstrapper-config.json`) for Runtime Bootstrapper. Managed by KIM. |
| controller-runtime | The Kubernetes controller framework (`sigs.k8s.io/controller-runtime`) used as the foundation for the webhook server, the Secret controller, and supporting infrastructure. |
| envtest | A controller-runtime testing utility that starts a real Kubernetes API server binary for integration tests, without requiring a live cluster. |
| `failurePolicy: Ignore` | A `MutatingWebhookConfiguration` setting that instructs the API server to admit the resource unchanged if the webhook cannot be reached. Runtime Bootstrapper uses this to remain non-blocking. |
| FIPS 140 | A US government standard for cryptographic modules. Runtime Bootstrapper is built with `GOFIPS140=v1.0.0` to produce a FIPS-140-compliant binary. |
| GetConfig | A function type (`func(context.Context) (*apiv1.Config, error)`) injected into the webhook defaulter and called on every admission request to fetch the current configuration. |
| Kyma Control Plane (KCP) | The central control plane for all Kyma runtimes. Hosts the shared resources (pull secret, ClusterTrustBundle, webhook ConfigMap) that are synchronized to individual runtimes. |
| Kyma Environment Broker (KEB) | The SAP BTP service that accepts Kyma runtime provisioning requests and creates Runtime CRs for KIM to act on. |
| Kyma Infrastructure Manager (KIM) | Installs and manages the lifecycle of Kyma runtimes and the components deployed on them, including Runtime Bootstrapper. |
| Kyma Lifecycle Manager (KLM) | Deploys Kyma modules into Kyma runtimes by managing Deployments, DaemonSets, and StatefulSets. Does not create bare Pods directly. |
| Kustomize | A Kubernetes configuration management tool used to generate environment-specific manifests (`config/default/` for production, `config/k3d/` for local development). |
| `KYMA_LANDSCAPE` | Environment variable injected into Pod containers by the `set-landscape` manipulation. Its value is the landscape identifier supplied to Runtime Bootstrapper using the `--landscape` flag at startup. |
| master secret | The `registry-credentials` Secret in `kyma-system`. The authoritative copy that the Secret controller reads and mirrors to all other namespaces. |
| `MutatingWebhookConfiguration` | The Kubernetes cluster-scoped resource (`rt-bootstrapper-mutating-webhook-configuration`) that registers Runtime Bootstrapper as a mutating webhook with the API server. |
| `namespaceFeatures` | A field in the `Config` struct that maps namespace names to a list of feature annotation keys. Pods in a listed namespace receive the named manipulations without requiring any annotation on the namespace or Pod. |
| opt-in annotation | An annotation of the form `rt-cfg.kyma-project.io/<feature>: "true"` placed on a namespace or Pod to enable a specific manipulation. |
| PodDefaulter | The function type `func(*corev1.Pod, map[string]string, *apiv1.Config) (bool, error)` implemented by each manipulation. Returns `true` if the Pod was modified. |
| rt-bootstrapper | The field manager name used for server-side apply patch operations by the Secret controller. |
| rt-bootstrapper-webhook | The field manager name used for server-side apply patch operations on the `MutatingWebhookConfiguration` by the certificate updater. |
| Runtime CR | A custom resource managed by KIM representing a single Kyma runtime instance. In the interim architecture, its labels are used as a signaling channel to trigger KIM reconciliation when shared resources change. |
| `SecretSyncInterval` | The duration after which the Secret controller re-queues itself to perform a full drift-correction sync of the pull secret across all namespaces. Default: 1 minute. |
| `set-landscape` | The feature annotation key `rt-cfg.kyma-project.io/set-landscape`; when activated, causes Runtime Bootstrapper to inject the `KYMA_LANDSCAPE` environment variable into every container. Requires the `--landscape` flag to be set at startup. |
| server-side apply | A Kubernetes API server feature where the client declares intent and the server computes and applies the diff, tracking field ownership. Used by the Secret controller and the certificate updater. |
| Kyma runtime | An instance of SAP BTP, Kyma runtime provisioned and managed by KIM. Runtime Bootstrapper runs inside each Kyma runtime. |
