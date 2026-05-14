# Solution Strategy

## Key Decisions

### Kubernetes Mutating Admission Webhook

Pod manipulation is implemented as a mutating admission webhook rather than as a controller that patches running Pods. The webhook intercepts a Pod at the earliest possible point – before `kubelet` ever sees the final spec – so the Pod starts correctly on the first attempt and no post-hoc remediation is needed for the normal path.

### Kubebuilder as the Foundation

The entire binary is using the [Kubebuilder framework](https://kubebuilder.io/) and is built on `sigs.k8s.io/controller-runtime`. This gives the system a production-grade controller loop (for Secret synchronization), webhook server infrastructure, leader election, health/readiness probes, and structured logging with minimal boilerplate.

### Opt-In Annotation Model with Three Layers

Manipulations are activated through a three-layer annotation precedence model: config-defined namespace defaults (highest priority, set by KIM), namespace annotations, and Pod annotations (lowest priority). For the full precedence rules and the `rt-cfg.kyma-project.io/all` alias, see [Cross-Cutting Concepts – Opt-In Annotation Model](01-08-crosscutting-concepts.md#opt-in-annotation-model).

This model means Kyma-managed namespaces (for example, `kyma-system` and `istio-system`) are handled silently because they are defined in the webhook configuration, while customers can self-service enable manipulations for their own namespaces or Pods.

### Configuration Re-Read Per Webhook Invocation

The webhook reads the `rt-bootstrapper-config` ConfigMap from the Kubernetes API on every admission request (using the `GetConfig` callback). This avoids the need to restart the webhook process after a configuration change and guarantees that the latest config is used without any cache invalidation logic.

### Self-Managed Certificate Rotation Using certwatcher

The webhook server uses `controller-runtime/certwatcher` to watch the on-disk TLS certificate files. When a renewed certificate is written (for example, by cert-manager), the watcher fires a callback that patches the **caBundle** field of the `MutatingWebhookConfiguration` automatically. No manual rotation step or Pod restart is required. Automated renewal of the certificate is handled by [Gardener's cert-management](https://github.com/gardener/cert-management).

### Secret Synchronization Using a Dedicated Controller

Rather than relying on KIM or an external tool to replicate the pull secret across namespaces, Runtime Bootstrapper includes its own controller that watches the master secret in `kyma-system` and mirrors it to all other namespaces. This keeps the component self-contained and ensures that a pull secret update propagates cluster-wide within the `SecretSyncInterval` (default: 1 minute) without external coordination.

### Single Binary, Two Components

Both the webhook server and the secret reconciler share one `cmd/main.go` entry point and one controller-manager process. This reduces operational overhead (one Deployment, one Service, one TLS certificate) while keeping the code strictly separated into `internal/webhook/` and `internal/controller/`.
