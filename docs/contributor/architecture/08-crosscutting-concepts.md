# Cross-Cutting Concepts

## Configuration Model

All runtime behavior of Runtime Bootstrapper is governed by a single JSON document stored in the ConfigMap `rt-bootstrapper-config` under the key `rt-bootstrapper-config.json`. The shape is:

```json
{
  "overrides": {
    "old.registry.example.com": "new.registry.example.com"
  },
  "clusterTrustBundle": {
    "name": "rt-bootstrapper-k3d.test:ctb:1",
    "volumeName": "rt-bootstrapper-certs",
    "volumeMountPath": "/etc/ssl/certs",
    "certWritePath": "kube-apiserver-serving.pem"
  },
  "availableFeatures": [
    "rt-cfg.kyma-project.io/alter-img-registry",
    "rt-cfg.kyma-project.io/add-img-pull-secret",
    "rt-cfg.kyma-project.io/add-cluster-trust-bundle",
    "rt-cfg.kyma-project.io/set-fips-mode"
  ],
  "namespaceFeatures": {
    "kyma-system": [
      "rt-cfg.kyma-project.io/alter-img-registry",
      "rt-cfg.kyma-project.io/add-img-pull-secret"
    ],
    "istio-system": ["rt-cfg.kyma-project.io/all"]
  }
}
```

`NewConfig` validates and cross-validates this document on every read. Unknown feature keys and `namespaceFeatures` entries that reference features not in `availableFeatures` are rejected with an error.

## Opt-In Annotation Model

Activating a manipulation for a Pod follows a three-layer precedence check. Any layer that matches causes the manipulation to be applied; all three are checked independently:

```
Priority order (highest to lowest):
  1. Config.namespaceFeatures[podNamespace]   ← set by KIM, not overridable
  2. Namespace annotation                     ← set by namespace owner
  3. Pod annotation                           ← set on the Pod template
```

The special annotation `rt-cfg.kyma-project.io/all: "true"` expands to all entries in `Config.AvailableFeatures` at evaluation time. Expansion happens in `Config.ExpandAnnotationAll()`.

## Idempotency

Every manipulation function is designed to be idempotent:
- **AlterImgRegistry**: only rewrites if the source registry matches a key in `overrides`; if the image was already rewritten, it will match the destination host, which has no entry in `overrides` and so is left unchanged.
- **AddImagePullSecrets**: uses `slices.Contains` to skip if the secret reference is already present.
- **AddClusterTrustBundle**: checks the existing volumes/volumeMounts by name and value equality before adding or replacing.
- **SetFipsMode**: checks each environment variable's current value before setting it.

The secret controller uses server-side apply (`client.Apply`) for all patches, which is inherently idempotent.

## TLS and Security

- The webhook server enforces TLS 1.3 as the minimum version (`tls.VersionTLS13`).
- HTTP/2 is disabled by default (`NextProtos: ["http/1.1"]`). It is recommended to keep HTTP/2 disabled; use `--enable-http2` only if required.
- The `caBundle` in the `MutatingWebhookConfiguration` is kept in sync with the on-disk CA certificate by the certificate rotation mechanism; the API server can always verify the webhook's certificate.
- The manager process never stores secrets in memory beyond the duration of a single reconcile loop invocation.

## Structured Logging

Runtime Bootstrapper uses the Go standard library `log/slog` for structured (key-value) logging throughout. The controller-runtime components additionally use `go-logr/logr` (bridged to zap via `zap.New(zap.UseFlagOptions(&opts))`). Log level defaults to `Debug`. Every reconcile invocation attaches a `uuid` field to its log entries for correlation.

## FIPS-140 Compliance

The binary is built with `GOFIPS140=v1.0.0` (enforced in the Makefile for all `go build`, `go run`, and `go test` invocations). This links the FIPS-140-validated Go crypto module and is required for deployment in regulated SAP BTP landscapes.

## Error Handling

- **Webhook panics**: `podCustomDefaulter.Default()` defers a `recover()` that converts any panic into a returned error, which causes the API server to reject the Pod admission with an informative error rather than crashing the process.
- **ConfigMap not found at startup**: The process exits with code 1 if the ConfigMap cannot be read at startup. This is intentional – a missing or invalid config is a misconfiguration that must be resolved before the webhook can safely operate.
- **Secret sync errors**: When patching a secret in one namespace fails, the error is recorded but the controller continues patching the remaining namespaces. All errors are collected and returned as a combined error, triggering controller-runtime's standard exponential backoff retry.
- **Certificate patch conflicts**: `BuildUpdateCABundle` wraps the patch in `retry.RetryOnConflict` with `retry.DefaultBackoff`.

## Testing Strategy

| Layer | Framework | Infrastructure |
|---|---|---|
| Unit tests | `testing` + `testify` | No Kubernetes; pure Go |
| Integration tests | Ginkgo v2 + Gomega | `envtest` (real API server binary, no cluster) |
| E2E tests | Ginkgo v2 + Gomega | Live k3d cluster with two local Docker registries (one authenticated, one open) |

The build tag `e2e` gates the E2E test files. The `test` Make target runs only the non-E2E subset.
