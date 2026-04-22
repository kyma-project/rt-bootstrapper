# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

All commands use `GOFIPS140=v1.0.0` when running Go toolchain steps. Tool binaries (kustomize, controller-gen, golangci-lint, setup-envtest) are downloaded into `./bin/` on first use.

```bash
# Build
make build          # generates manifests + code, then builds bin/manager

# Test (unit + integration via envtest, excludes e2e)
make test

# Run a single test package
KUBEBUILDER_ASSETS="$(./bin/setup-envtest use --bin-dir bin -p path)" \
  go test ./internal/webhook/v1/... -run TestFoo -v

# Lint
make lint           # runs golangci-lint
make lint-fix       # auto-fixes linting issues

# Code generation (must run after editing kubebuilder markers)
make generate       # DeepCopy methods
make manifests      # RBAC, webhook, CRD manifests

# E2E tests (requires k3d)
make test-e2e       # creates k3d cluster + docker registry, then runs ./test/e2e/

# Deploy to local k3d cluster
k3d cluster create test-cluster
make deploy

# Tear down e2e cluster + registries
make cleanup-test-e2e
```

## Architecture

RT Bootstrapper is a Kubernetes operator with two active components sharing a single binary (`cmd/main.go`):

### 1. Mutating Admission Webhook (`internal/webhook/`)

Intercepts `Pod` creation requests from the API server. Webhook is opt-in — a Pod is modified only when one of the following is true:
- The `pkg/api/v1.Config` (loaded from a ConfigMap `rt-bootstrapper-config`) declares `namespaceFeatures` for the Pod's namespace.
- The namespace has a feature annotation (`rt-cfg.kyma-project.io/*`).
- The Pod itself has a feature annotation.

**Feature annotations** (`pkg/api/v1/types.go`):
| Annotation | Effect |
|---|---|
| `rt-cfg.kyma-project.io/alter-img-registry` | Rewrites container image registries per `Config.Overrides` |
| `rt-cfg.kyma-project.io/add-img-pull-secret` | Injects imagePullSecret into Pod spec |
| `rt-cfg.kyma-project.io/add-cluster-trust-bundle` | Projects a ClusterTrustBundle as a volume |
| `rt-cfg.kyma-project.io/set-fips-mode` | Sets `KYMA_FIPS_MODE_ENABLED` and `FIPS_MODE_ENABLED` env vars |
| `rt-cfg.kyma-project.io/all` | Alias that expands to all features listed in `Config.AvailableFeatures` |

**`PodDefaulter` pattern** (`internal/webhook/v1/pod_defaulters.go`): Each feature is a `func(pod, nsAnnotations, cfg) (bool, error)` built by a `BuildXxx()` constructor. The `podCustomDefaulter.Default()` method calls each active defaulter in sequence, reads the config fresh per call via `GetConfig`, and marks modified pods with `rt-bootstrapper.kyma-project.io/modified: "true"`.

**Certificate rotation** (`internal/webhook/server/server.go`, `internal/webhook/certificate/`): The webhook server wraps `certwatcher` to hot-reload TLS certs and automatically patches the `caBundle` field of the `MutatingWebhookConfiguration` on cert renewal.

### 2. Secret Controller (`internal/controller/secret_controller.go`)

Watches a "master" image-pull secret in `kyma-system` and ensures copies exist in every namespace. Three reconcile scenarios (distinguished by `ctrl.Request`):

- **Master secret updated** → syncs to all namespaces, re-queues after `SecretSyncInterval`.
- **Per-namespace secret updated/deleted** → force-patches that single namespace's copy.
- **New namespace created** → patches a copy into the new namespace.

The controller uses two predicates to filter events:
- `createNsPredicate` — passes only Namespace create events (ignoring the master namespace).
- `masterSecret` predicate — passes only events on the named master secret.

### Configuration (`pkg/api/v1/types.go`)

`Config` is a JSON document stored in a ConfigMap under the key `rt-bootstrapper-config.json`. Key fields:
- `overrides`: registry hostname rewrites (old → new).
- `clusterTrustBundle`: ClusterTrustBundle name and volume mount config.
- `namespaceFeatures`: per-namespace list of feature annotation keys (or `rt-cfg.kyma-project.io/all`).
- `availableFeatures`: allowlist of features that can be activated; must be a subset of `KnownFeatureKeys`.

`NewConfig` validates the struct and cross-validates `namespaceFeatures` against `availableFeatures`. The config is re-read from the cluster on every webhook invocation (via `GetConfig` callback).

### Testing

- Unit/integration tests use `envtest` (real Kubernetes API server, no cluster needed).
- E2E tests (`test/e2e/`, build tag `e2e`) require a live k3d cluster with two local Docker registries (one with auth, one open) set up by `hack/rt-e2e-tests/setup-k3d-registry-bootstrapper.sh`.
- Test suites use Ginkgo v2 + Gomega.
