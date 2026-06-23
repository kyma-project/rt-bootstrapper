# Runtime Bootstrapper

## What Is the Runtime Bootstrapper?

The Runtime Bootstrapper is a component that automatically adapts your workloads when they are deployed on SAP BTP, Kyma runtime. It runs transparently in the background and adjusts your Pods — and only your Pods — to meet the requirements of the specific landscape you are deploying into (for example, a landscape that uses a private container registry, requires FIPS-compliant workloads, or uses custom TLS certificates).

You do not need to change your application code. All adjustments happen at Pod creation time, before the Pod starts running.

> **Important:** The Runtime Bootstrapper only modifies **Pods**. It does not touch any other Kubernetes resources such as Deployments, Services, ConfigMaps, or Secrets.

---

## How to Enable It

You can opt in to Runtime Bootstrapper features at two levels:

### Option 1: Annotate the Namespace

Add one or more feature annotations to your Namespace. All Pods created in that namespace will automatically receive the corresponding adjustments — no changes to individual workload manifests are needed.

```yaml
apiVersion: v1
kind: Namespace
metadata:
  name: my-namespace
  annotations:
    rt-cfg.kyma-project.io/alter-img-registry: "true"
    rt-cfg.kyma-project.io/add-img-pull-secret: "true"
```

### Option 2: Annotate the Pod (or Pod Template)

Add annotations directly to your Pod or to the `spec.template.metadata.annotations` section of a Deployment, StatefulSet, or similar resource. Only that specific Pod (or Pods from that template) will be adjusted.

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-app
spec:
  template:
    metadata:
      annotations:
        rt-cfg.kyma-project.io/alter-img-registry: "true"
        rt-cfg.kyma-project.io/add-img-pull-secret: "true"
    spec:
      containers:
        - name: my-container
          image: my-registry.example.com/my-app:1.0.0
```

Both levels can be combined. If a feature is enabled on the namespace, all Pods in that namespace benefit from it regardless of their own annotations.

> **Tip:** To enable all available features at once, use the shorthand annotation `rt-cfg.kyma-project.io/all: "true"` on either the namespace or the Pod.

---

## Supported Annotations

Each annotation enables one specific feature. Set the value to `"true"` to activate it.

---

### `rt-cfg.kyma-project.io/alter-img-registry`

**What it does:** Rewrites the container registry host in image references.

Some landscapes require that container images are pulled from a private or landscape-specific registry instead of the original public registry. When this feature is enabled, the Runtime Bootstrapper automatically rewrites the registry hostname in the `image` field of every container and init-container in your Pod.

**What changes in your Pod:**

- `.spec.containers[*].image` — registry hostname is replaced
- `.spec.initContainers[*].image` — registry hostname is replaced

You do not need to know the target registry address; this is configured centrally for the landscape.

---

### `rt-cfg.kyma-project.io/add-img-pull-secret`

**What it does:** Injects image pull credentials into your Pod.

When a private container registry requires authentication, the Runtime Bootstrapper adds a reference to the landscape's image pull Secret (`registry-credentials`) to your Pod. This ensures your Pod can pull images without you having to manage registry credentials yourself.

**What changes in your Pod:**

- `.spec.imagePullSecrets[]` — entry `registry-credentials` is appended

If the Secret reference is already present, it is not added again.

---

### `rt-cfg.kyma-project.io/add-cluster-trust-bundle`

**What it does:** Mounts the cluster's TLS certificate bundle into your containers.

Some landscapes use custom TLS certificates that are not included in the standard operating system trust store. When this feature is enabled, the Runtime Bootstrapper mounts the cluster's certificate bundle as a read-only volume into every container (including init-containers) under the path `/etc/ssl/certs`. This allows your application to trust landscape-specific HTTPS endpoints without any code changes.

**What changes in your Pod:**

- `.spec.volumes[]` — a projected volume named `rt-bootstrapper-certs` is added
- `.spec.containers[*].volumeMounts` — the volume is mounted read-only at `/etc/ssl/certs`
- `.spec.initContainers[*].volumeMounts` — same mount applied to init-containers

> **Important — certificate rotation:** The mounted CA bundle can change over time. Certificates are rotated periodically for security reasons, and the file at `/etc/ssl/certs` will be updated in place without restarting your Pod. Most application runtimes load TLS trust stores once at startup and do not automatically pick up changes from the filesystem. If your application is not aware of this, it will start rejecting HTTPS connections to landscape endpoints after a rotation event.
>
> To handle certificate rotation, choose one of the following approaches:
>
> - **Watch for file changes inside your application.** Implement a file watcher that detects changes under `/etc/ssl/certs` and reloads the trust store at runtime without restarting the process. This is the most resilient approach and avoids any downtime.
> - **Trigger an externally managed restart.** Use a controller that watches the underlying `ClusterTrustBundle` resource and automatically rolls your workload when it changes. This is conceptually similar to how [stakater/Reloader](https://github.com/stakater/Reloader) restarts Pods when a referenced ConfigMap or Secret changes. A restart ensures the new certificate is loaded, at the cost of a brief interruption.

---

### `rt-cfg.kyma-project.io/set-fips-mode`

**What it does:** Signals FIPS 140 compliance mode to your workloads.

FIPS (Federal Information Processing Standards) mode restricts cryptographic operations to approved algorithms. When this feature is enabled, the Runtime Bootstrapper sets two environment variables in every container and init-container. Your application can read these variables to activate its own FIPS-compliant code paths.

**What changes in your Pod:**

- `.spec.containers[*].env[]` — the following variables are added to every container:
  - `KYMA_FIPS_MODE_ENABLED=true`
  - `FIPS_MODE_ENABLED=true` *(legacy compatibility)*
- `.spec.initContainers[*].env[]` — same variables added to init-containers

---

### `rt-cfg.kyma-project.io/set-landscape`

**What it does:** Injects the landscape identifier into your containers.

Some applications need to know which landscape they are running on to adjust their behavior (for example, to point to the correct backend endpoint or to display the correct region label). When this feature is enabled, the Runtime Bootstrapper injects the landscape identifier as an environment variable into every container and init-container.

**What changes in your Pod:**

- `.spec.containers[*].env[]` — the following variable is added:
  - `KYMA_LANDSCAPE=<landscape-identifier>`
- `.spec.initContainers[*].env[]` — same variable added to init-containers

The actual value of the landscape identifier is provided by the landscape operator and is not configurable by the workload owner.

---

### `rt-cfg.kyma-project.io/all`

**What it does:** Shorthand to enable all available features at once.

Setting this annotation to `"true"` is equivalent to setting every feature annotation listed above. Use it when you want your namespace or Pod to receive all landscape adaptations without listing them individually.

```yaml
annotations:
  rt-cfg.kyma-project.io/all: "true"
```

---

## Confirming a Pod Was Modified

After the Runtime Bootstrapper processes a Pod, it adds the following annotation to the Pod metadata:

```
rt-bootstrapper.kyma-project.io/modified: "true"
```

You can verify this with:

```sh
kubectl get pod <pod-name> -o jsonpath='{.metadata.annotations.rt-bootstrapper\.kyma-project\.io/modified}'
```

If the output is `true`, the Pod was successfully processed.

---

## Summary Table

| Annotation | What it enables |
|---|---|
| `rt-cfg.kyma-project.io/alter-img-registry: "true"` | Rewrite container image registry hostnames |
| `rt-cfg.kyma-project.io/add-img-pull-secret: "true"` | Inject image pull Secret (`registry-credentials`) |
| `rt-cfg.kyma-project.io/add-cluster-trust-bundle: "true"` | Mount cluster TLS certificates at `/etc/ssl/certs` |
| `rt-cfg.kyma-project.io/set-fips-mode: "true"` | Set `KYMA_FIPS_MODE_ENABLED` and `FIPS_MODE_ENABLED` env vars |
| `rt-cfg.kyma-project.io/set-landscape: "true"` | Set `KYMA_LANDSCAPE` env var |
| `rt-cfg.kyma-project.io/all: "true"` | Enable all of the above |
