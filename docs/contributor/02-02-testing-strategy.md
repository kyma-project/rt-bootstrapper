# Testing Strategy

Runtime Bootstrapper uses three testing layers, each targeting a different level of the system. All layers are run with `GOFIPS140=v1.0.0` enforced by the Makefile to ensure FIPS-140-compliant binaries are produced during testing.

## Layers at a Glance

| Layer | Framework | Infrastructure | Build tag |
|---|---|---|---|
| Unit | `testing` + `testify` | None — pure Go, no Kubernetes | (none) |
| Integration | Ginkgo v2 + Gomega | `envtest` — real API server binary, no cluster | (none) |
| E2E | Ginkgo v2 + Gomega | Live k3d cluster with two local Docker registries | `e2e` |

## Running the Tests

| Command | What it runs |
|---|---|
| `make test` | Unit and integration tests; excludes the `test/e2e/` package; writes coverage to `cover.out` |
| `make test-e2e` | Full E2E suite; provisions k3d cluster, installs Calico and cert-manager, sets up registries, then runs the tests |
| `make setup-test-e2e` | Provisions the k3d cluster only, without running tests |
| `make cleanup-test-e2e` | Tears down the k3d cluster and stops the local Docker registries |
| `make setup-envtest` | Downloads the `envtest` binaries required for integration tests |

---

## Unit Tests

**Location:** `internal/webhook/certificate/`, `internal/webhook/k8s/`, `pkg/api/v1/`, `internal/webhook/v1/pod_defaulter_landscape_test.go`

Unit tests cover pure logic with no Kubernetes client dependency:

- **`pkg/api/v1/`** — Config struct parsing, validation, and cross-validation of `availableFeatures` vs `namespaceFeatures`.
- **`internal/webhook/k8s/`** — Stateless helpers: image registry rewriting (`AlterPodImageRegistry`) and annotation matching (`Contains`).
- **`internal/webhook/certificate/`** — The `caBundle` patch callback logic.
- **`internal/webhook/v1/pod_defaulter_landscape_test.go`** — The `SetLandscape` manipulation (pure function, no API server).

---

## Integration Tests

**Location:** `internal/controller/`, `internal/webhook/server/`, `internal/webhook/v1/`

Integration tests run against a real Kubernetes API server binary provided by `envtest`. No cluster is required, but `envtest` binaries must be downloaded first with `make setup-envtest`.

The `ENVTEST_K8S_VERSION` is derived automatically from the `k8s.io/api` module version pinned in `go.mod`.

**What is tested:**

- **`internal/webhook/v1/`** — Full webhook dispatch loop: annotation evaluation across all three layers, each `PodDefaulter` applied in combination, and idempotency (calling `Default()` twice on the same Pod).
- **`internal/controller/`** — `SecretReconciler`: master Secret synced to new namespaces, master Secret update propagated to all namespaces, predicate filtering (`createNsPredicate`, `masterSecret`).
- **`internal/webhook/server/`** — TLS server startup and readiness probe behaviour.

---

## E2E Tests

**Location:** `test/e2e/`

**Build tag:** `e2e` — E2E test files are excluded from `make test` and only compiled when `-tags=e2e` is passed.

E2E tests run the fully built manager image against a real k3d cluster with two local Docker registries: one open (unauthenticated) and one secured (requires pull-Secret authentication). They validate the golden path for each manipulation and the full opt-in annotation model.

### Prerequisites

| Requirement | Version / Detail |
|---|---|
| k3d | Any recent version |
| Kubernetes | `rancher/k3s:v1.33.6-k3s1` (set via `K3D_IMAGE`) |
| API server feature gates | `ClusterTrustBundle=true,ClusterTrustBundleProjection=true` (set via `K3D_ARGS`) |
| CNI | Calico v3.31.3 — required for `NetworkPolicy` enforcement |
| cert-manager | Installed automatically; skip with `CERT_MANAGER_INSTALL_SKIP=true` if already present |
| Docker registries | Two local registries provisioned by `make setup-docker-registry` |

### Environment Variables

| Variable | Default | Effect |
|---|---|---|
| `K3D_CLUSTER` | `rt-bootstrapper-test-e2e` | Name of the k3d cluster to create or reuse |
| `K3D_IMAGE` | `rancher/k3s:v1.33.6-k3s1` | k3s image used for the cluster |
| `K3D_ARGS` | *(see Makefile)* | Extra arguments passed to `k3d cluster create`, including the feature gate flags |
| `CERT_MANAGER_INSTALL_SKIP` | `false` | Set to `true` to skip cert-manager installation if it is already present on the cluster |
