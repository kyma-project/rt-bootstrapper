# Adapting Your Workloads to Kyma Landscape Requirements

Learn how to use Runtime Bootstrapper, a built-in Kyma component, to automatically adapt your Pods to landscape-specific requirements such as private registries, custom TLS certificates, and FIPS mode signaling.

> ### Note:
> The content in this topic is only relevant for China (Shanghai) and Government Cloud (US) regions.

When you deploy workloads on SAP BTP, Kyma runtime in certain regions, such as China (Shanghai) and Government Cloud (US), Pods need specific configurations. For example, Pods may need to pull images from a private registry or trust landscape-specific TLS certificates. Runtime Bootstrapper automatically applies these configurations to your Pods at creation time. This process is transparent and requires no changes to your application code.

Runtime Bootstrapper only modifies Pods. It doesn't affect any other Kubernetes resources, such as Deployments, Services, ConfigMaps, or Secrets.

## Enabling Runtime Bootstrapper

To enable Runtime Bootstrapper features, annotate either a namespace or a Pod (or Pod template).

### Annotating a Namespace

Add one or more feature annotations to your namespace. All Pods created in that namespace then receive the corresponding adjustments automatically. You don't need to change individual workload manifests.

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

Add annotations directly to your Pod or to the `spec.template.metadata.annotations` section of a Deployment, StatefulSet, or a similar resource. As a result, only that specific Pod or Pods from that template are adjusted.

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

You can combine both methods. If you enable a feature on a namespace, all Pods in that namespace benefit from it. There is no way to opt out at the Pod level.

## Supported Annotations

Each annotation enables one specific feature. To activate it, set the value to `"true"`.

The availability shown below may change. To confirm a feature is active in your landscape, see [Confirming a Pod Was Modified](#confirming-a-pod-was-modified). If a feature appears inactive despite being listed as available, contact your landscape operator.

| Annotation                                                                                         | Typical use case                                                                                    | Available for Government Cloud (US) region                                                                                                                                                                                                                                                | Available for China (Shanghai) region |
|----------------------------------------------------------------------------------------------------|-----------------------------------------------------------------------------------------------------|-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|---------------------------------------|
| [`rt-cfg.kyma-project.io/add-cluster-trust-bundle`](#rt-cfgkyma-projectioadd-cluster-trust-bundle) | Mounts custom CA certificates required to trust SAP backend endpoints                               | Yes, it's a primary feature. The CA bundle is managed by the landscape operator team. If TLS communication fails due to missing or outdated certificates, create a support ticket. See [Getting Support](https://help.sap.com/docs/btp/sap-business-technology-platform/getting-support). | No                                    |
| [`rt-cfg.kyma-project.io/add-img-pull-secret`](#rt-cfgkyma-projectioadd-img-pull-secret)           | Only relevant when pulling images from a private registry that requires authentication              | No                                                                                                                                                                                                                                                                                        | Yes                                   |
| [`rt-cfg.kyma-project.io/alter-img-registry`](#rt-cfgkyma-projectioalter-img-registry)             | Only required when the container registry hostname must be rewritten to a landscape-specific mirror | Yes                                                                                                                                                                                                                                                                                       | Yes                                   |
| [`rt-cfg.kyma-project.io/set-fips-mode`](#rt-cfgkyma-projectioset-fips-mode)                       | Signals to workloads that they must use FIPS 140-compliant cryptography                            | Yes                                                                                                                                                                                                                                                                                       | No                                    |
| [`rt-cfg.kyma-project.io/set-landscape`](#rt-cfgkyma-projectioset-landscape)                       | Injects the landscape identifier for workloads that need to know which environment they run in      | No                                                                                                                                                                                                                                                                                        | No                                    |
<!-- The set-landscape annotation is only documented in the open-source documentation. -->

> ### Tip:
> To enable all features available in your landscape at once, use the shorthand annotation [`rt-cfg.kyma-project.io/all: "true"`](#rt-cfgkyma-projectioall) on either a namespace or a Pod.

### `rt-cfg.kyma-project.io/add-cluster-trust-bundle`

The annotation mounts the cluster's TLS certificate bundle into your containers.

Some landscapes use custom TLS certificates that are not included in the standard operating system trust store. When you enable this feature, Runtime Bootstrapper mounts the cluster's certificate bundle as a read-only volume into every container (including init-containers) under the path `/etc/ssl/certs`. As a result, your application can trust landscape-specific HTTPS endpoints without any code changes.

The annotation causes the following changes in your Pod:

- `.spec.volumes[]` - a projected volume named `rt-bootstrapper-certs` is added
- `.spec.containers[*].volumeMounts` - the volume is mounted read-only at `/etc/ssl/certs`
- `.spec.initContainers[*].volumeMounts` - same mount applied to init-containers

#### Certificate Rotation

> ### Caution:
> The mounted CA bundle is rotated periodically for security, and the file at `/etc/ssl/certs` is updated in place without restarting your Pod. Because most applications load TLS trust stores only at startup, your application might not automatically use the new certificates after a rotation. This can cause TLS connection failures.

To handle certificate rotation, choose one of the following approaches:

- Watch for file changes inside your application:  Implement a file watcher that detects changes under `/etc/ssl/certs` and reloads the trust store at runtime without restarting the process. This is the most resilient approach and avoids any downtime.
- Trigger an externally managed restart: Use a controller that watches the underlying `ClusterTrustBundle` resource and automatically rolls your workload when it changes. This is conceptually similar to how [stakater/Reloader](https://github.com/stakater/Reloader) restarts Pods when a referenced ConfigMap or Secret changes. A restart ensures the new certificate is loaded, at the cost of a brief interruption.

### `rt-cfg.kyma-project.io/add-img-pull-secret`

The annotation injects image pull credentials into your Pod.

When a private container registry requires authentication, Runtime Bootstrapper adds a reference to the landscape's image pull Secret (`registry-credentials`) to your Pod. Your Pod can then pull images - you don't have to manage registry credentials yourself.

The annotation causes the following change in your Pod:

- `.spec.imagePullSecrets[]` - the `registry-credentials` entry is appended

If the Secret reference is already present, it is not added again.

### `rt-cfg.kyma-project.io/alter-img-registry`

The annotation rewrites the container registry host in image references.

Some landscapes require container images to be pulled from a private or landscape-specific registry rather than the original public registry. When you enable this feature, Runtime Bootstrapper automatically rewrites the registry hostname in the **image** field of every container and init-container in your Pod. You don't need to know the target registry address, because it's configured centrally for the landscape.

The annotation causes the following changes in your Pod:

- `.spec.containers[*].image` - registry hostname is replaced
- `.spec.initContainers[*].image` - registry hostname is replaced

### `rt-cfg.kyma-project.io/set-fips-mode`

The annotation signals FIPS 140 compliance mode to your workloads.

Federal Information Processing Standards (FIPS) mode restricts cryptographic operations to approved algorithms. When you enable this feature, Runtime Bootstrapper sets two environment variables in every container and init-container. Your application can read these variables to activate its own FIPS-compliant code paths.

The annotation causes the following changes in your Pod:

- `.spec.containers[*].env[]` - the following variables are added to every container:
  - `KYMA_FIPS_MODE_ENABLED=true`
  - `FIPS_MODE_ENABLED=true` (legacy compatibility)
- `.spec.initContainers[*].env[]` - the following variables are added to init-containers:
  - `KYMA_FIPS_MODE_ENABLED=true`
  - `FIPS_MODE_ENABLED=true` (legacy compatibility)

<!-- The set-landscape annotation is only documented in the open-source documentation. -->
### `rt-cfg.kyma-project.io/set-landscape`

The annotation injects the landscape identifier into your containers.

Some applications must know which landscape they are running on to adjust their behavior (for example, to point to the correct backend endpoint or to display the correct region label). When you enable this feature, Runtime Bootstrapper injects the landscape identifier as an environment variable into every container and init-container.

The annotation causes the following changes in your Pod:

- `.spec.containers[*].env[]` - the `KYMA_LANDSCAPE=<landscape-identifier>` variable is added
- `.spec.initContainers[*].env[]` - the `KYMA_LANDSCAPE=<landscape-identifier>` variable is added

The actual value of the landscape identifier is provided by the landscape operator and is not configurable by the workload owner.

### `rt-cfg.kyma-project.io/all`

The annotation enables all features available for your region at once.

Setting this annotation to `"true"` applies all features available for your region (see [Supported Annotations](#supported-annotations)) that are enabled in the operator's configuration. Use it when you want your namespace or Pod to receive all applicable landscape adaptations without listing them individually.

```yaml
annotations:
  rt-cfg.kyma-project.io/all: "true"
```

## Confirming a Pod Was Modified

After Runtime Bootstrapper processes a Pod, it adds the following annotation to the Pod metadata:

```
rt-bootstrapper.kyma-project.io/modified: "true"
```

To verify this, replace the placeholder and run:

```sh
kubectl get pod <pod-name> -o jsonpath='{.metadata.annotations.rt-bootstrapper\.kyma-project\.io/modified}'
```

If the output is `true`, Runtime Bootstrapper applied at least one modification when the Pod was created. This annotation reflects the state at creation time only. If the landscape configuration changes later, recreated Pods might silently receive fewer modifications without automatic notification. If you suspect a feature is no longer active, contact your landscape operator and check the Runtime Bootstrapper logs for relevant warnings:

```sh
kubectl logs -n kyma-system -l app.kubernetes.io/name=rt-bootstrapper
```

If the output is empty or `false`, either no feature annotations were set on the Pod or its namespace, or the annotations used are not supported in this landscape. See [Supported Annotations](#supported-annotations) for details.
