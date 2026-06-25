# Runtime Bootstrapper

## What Is Runtime Bootstrapper?

Runtime Bootstrapper is a component that automatically adapts your workloads when they are deployed on SAP BTP, Kyma runtime. It runs transparently in the background and adjusts your Pods — and only your Pods — to meet the requirements of the specific landscape you are deploying into (for example, a landscape that uses a private container registry, requires FIPS-compliant workloads, or uses custom TLS certificates).

You do not need to change your application code. All adjustments happen at Pod creation time, before the Pod starts running.

> ### Note:
> Runtime Bootstrapper only modifies Pods. It does not touch any other Kubernetes resources, such as Deployments, Services, ConfigMaps, or Secrets.

---

## Enabling Runtime Bootstrapper

You can opt in to Runtime Bootstrapper features at two levels:
- Annotating a namespace
- Annotating a Pod or a Pod template

### Annotating a Namespace

Add one or more feature annotations to your namespace. As a result, all Pods created in that namespace automatically receive the corresponding adjustments, without requiring changes to individual workload manifests.

```yaml
apiVersion: v1
kind: Namespace
metadata:
  name: my-namespace
  annotations:
    rt-cfg.kyma-project.io/alter-img-registry: "true"
    rt-cfg.kyma-project.io/add-img-pull-secret: "true"
```

### Annotating a Pod or a Pod Template

Add annotations directly to your Pod or to the `spec.template.metadata.annotations` section of a Deployment, StatefulSet, or similar resource. As a result, only that specific Pod or Pods from that template are adjusted.

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

Both levels can be combined. If a feature is enabled on a namespace, all Pods in that namespace benefit from it regardless of their own annotations.

---

## Supported Annotations

Each annotation enables one specific feature. To activate it, set the value to `"true"`.

| Annotation | Typical use case |
|---|---|
| [`rt-cfg.kyma-project.io/add-cluster-trust-bundle`](#rt-cfgkyma-projectioadd-cluster-trust-bundle) | Primary feature for NS2 (Sovereign Cloud) — mounts custom CA certificates required to trust SAP backend endpoints |
| [`rt-cfg.kyma-project.io/add-img-pull-secret`](#rt-cfgkyma-projectioadd-img-pull-secret) | Only relevant when pulling images from a private registry that requires authentication |
| [`rt-cfg.kyma-project.io/alter-img-registry`](#rt-cfgkyma-projectioalter-img-registry) | Only required when the container registry hostname must be rewritten to a landscape-specific mirror |
| [`rt-cfg.kyma-project.io/set-fips-mode`](#rt-cfgkyma-projectioset-fips-mode) | Primarily NS2-related — signals to workloads that FIPS 140-compliant cryptography should be used |
| [`rt-cfg.kyma-project.io/set-landscape`](#rt-cfgkyma-projectioset-landscape) | Injects the landscape identifier for workloads that need to know which environment they run in |

> ### Tip:
> To enable all available features at once, use the shorthand annotation [`rt-cfg.kyma-project.io/all: "true"`](#rt-cfg.kyma-project.io/all) on either a namespace or a Pod.

> ## Note
> In the US Sovereign Cloud (NS2) environment, the CA bundle injection (rt-cfg.kyma-project.io/add-cluster-trust-bundle) is the primary feature you need. The other features — image registry rewriting, pull secret injection, FIPS mode, and landscape identification — are only required in special cases and can be ignored for common deployments.

---

### `rt-cfg.kyma-project.io/alter-img-registry`

This annotation rewrites the container registry host in image references.

Some landscapes require container images to be pulled from a private or landscape-specific registry rather than the original public registry. When this feature is enabled, Runtime Bootstrapper automatically rewrites the registry hostname in the **image** field of every container and init-container in your Pod.

The annotation causes the following changes in your Pod:

- `.spec.containers[*].image` — registry hostname is replaced
- `.spec.initContainers[*].image` — registry hostname is replaced

You don't need to know the target registry address, because it's configured centrally for the landscape.

---

### `rt-cfg.kyma-project.io/add-img-pull-secret`

This annotation injects image pull credentials into your Pod.

When a private container registry requires authentication, Runtime Bootstrapper adds a reference to the landscape's image pull Secret (`registry-credentials`) to your Pod. This ensures your Pod can pull images without you having to manage registry credentials yourself.

The annotation causes the following change in your Pod:

- `.spec.imagePullSecrets[]` — entry `registry-credentials` is appended

If the Secret reference is already present, it is not added again.

---

### `rt-cfg.kyma-project.io/add-cluster-trust-bundle`

The annotation mounts the cluster's TLS certificate bundle into your containers.

Some landscapes use custom TLS certificates that are not included in the standard operating system trust store. When this feature is enabled, Runtime Bootstrapper mounts the cluster's certificate bundle as a read-only volume into every container (including init-containers) under the path `/etc/ssl/certs`. With this, your application can trust landscape-specific HTTPS endpoints without any code changes.

> **CA bundle ownership (NS2):** The CA bundle is managed by the NS2 operator team, not by individual application teams. If TLS communication between your workloads and SAP backends fails due to missing or outdated certificates, you must involve the NS2 operator team to issue a suitable replacement certificate. Once a new certificate is available, the Kyma team updates the CA bundle in its configuration so it becomes accessible on Kyma runtimes.
>
> **Hint:** Certificate changes must be handled via a service request. Make sure to inform the Kyma SRE team about any such changes so they can update the CA bundle configuration on the Kyma runtimes in a timely manner.

The annotation causes the following changes in your Pod:

- `.spec.volumes[]` — a projected volume named `rt-bootstrapper-certs` is added
- `.spec.containers[*].volumeMounts` — the volume is mounted read-only at `/etc/ssl/certs`
- `.spec.initContainers[*].volumeMounts` — same mount applied to init-containers

#### Certificate Rotation

> ### Caution:
> The mounted CA bundle can change over time. Certificates are rotated periodically for security reasons, and the file at `/etc/ssl/certs` is updated in place without restarting your Pod. Most application runtimes load TLS trust stores once at startup and do not automatically pick up changes from the filesystem. If your application is not aware of this, it starts rejecting HTTPS connections to landscape endpoints after a rotation event.

To handle certificate rotation, choose one of the following approaches:

- Watch for file changes inside your application:  Implement a file watcher that detects changes under `/etc/ssl/certs` and reloads the trust store at runtime without restarting the process. This is the most resilient approach and avoids any downtime.
- Trigger an externally managed restart: Use a controller that watches the underlying `ClusterTrustBundle` resource and automatically rolls your workload when it changes. This is conceptually similar to how [stakater/Reloader](https://github.com/stakater/Reloader) restarts Pods when a referenced ConfigMap or Secret changes. A restart ensures the new certificate is loaded, at the cost of a brief interruption.

---

### `rt-cfg.kyma-project.io/set-fips-mode`

This annotation signals FIPS 140 compliance mode to your workloads.

Federal Information Processing Standards (FIPS) mode restricts cryptographic operations to approved algorithms. When this feature is enabled, Runtime Bootstrapper sets two environment variables in every container and init-container. Your application can read these variables to activate its own FIPS-compliant code paths.

The annotation causes the following changes in your Pod:

- `.spec.containers[*].env[]` — the following variables are added to every container:
  - `KYMA_FIPS_MODE_ENABLED=true`
  - `FIPS_MODE_ENABLED=true` *(legacy compatibility)*
- `.spec.initContainers[*].env[]` — the following variables are added to init-containers:
  - `KYMA_FIPS_MODE_ENABLED=true`
  - `FIPS_MODE_ENABLED=true` *(legacy compatibility)*

---

### `rt-cfg.kyma-project.io/set-landscape`

This annotation injects the landscape identifier into your containers.

Some applications must know which landscape they are running on to adjust their behavior (for example, to point to the correct backend endpoint or to display the correct region label). When this feature is enabled, Runtime Bootstrapper injects the landscape identifier as an environment variable into every container and init-container.

The annotation causes the following changes in your Pod:

- `.spec.containers[*].env[]` - the `KYMA_LANDSCAPE=<landscape-identifier>` variable is added
- `.spec.initContainers[*].env[]` - the `KYMA_LANDSCAPE=<landscape-identifier>` variable is added

The actual value of the landscape identifier is provided by the landscape operator and is not configurable by the workload owner.

---

### `rt-cfg.kyma-project.io/all`

The annotation is a shorthand to enable all available features at once.

Setting this annotation to `"true"` is equivalent to setting every feature annotation listed above. Use it when you want your namespace or Pod to receive all landscape adaptations without listing them individually.

```yaml
annotations:
  rt-cfg.kyma-project.io/all: "true"
```

---

## Confirming a Pod Was Modified

After Runtime Bootstrapper processes a Pod, it adds the following annotation to the Pod metadata:

```
rt-bootstrapper.kyma-project.io/modified: "true"
```

To verify this, replace the placeholder and run:

```sh
kubectl get pod <pod-name> -o jsonpath='{.metadata.annotations.rt-bootstrapper\.kyma-project\.io/modified}'
```

If the output is `true`, the Pod was successfully processed.

