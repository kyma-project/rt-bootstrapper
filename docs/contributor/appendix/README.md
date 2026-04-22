
# Runtime Bootstrapper


## Overview
Kyma landscapes often require unique infrastructure setups, such as private container registries, certificate-based access mechanisms, or other specialized configurations tailored to specific contexts or markets. These setups make each Kyma landscape distinct.

Kyma modules, by default, are not designed to accommodate these landscape-specific differences. Without adjustments, they may face functional limitations, incompatibilities, or fail to operate within such landscapes.

To address this challenge, Runtime Bootstrapper applies landscape-specific configurations to Kyma modules and the workloads the configurations install. It ensures compatibility and functionality across diverse landscapes.

Runtime Bootstrapper is implemented as a mutating webhook that intercepts `create` or `update` requests for Pods before they are applied by Kubernetes `kubelet`. It modifies or rewrites parts of the Pod manifests to align them with the landscape requirements.

> [!NOTE]
> The webhook intercepts only Pods. Other resources, such as Deployments, DaemonSets, or StatefulSets, are ignored.

## Pod Manipulations

Runtime Bootstrapper modifies a Pod only if one of the following conditions is met:

1. The Pod runs within a namespace listed in the webhook's default configuration. All Pods in such namespaces are automatically intercepted and modified. This option is primarily used for Kyma-managed namespaces (e.g., `kyma-system`, `istio-system`, etc.).
2. The namespace contains an annotation indicating that Pods within the namespace should be intercepted.
3. The Pod itself is annotated to be intercepted by the webhook.

### Applied Manipulations

The table below provides an overview of the different manipulations supported by Runtime Bootstrapper.

The **Opt-In Annotation** column contains the annotation that must be added to a namespace or Pod to enable the webhook manipulation for it. The annotation is only required if the Pod is **not** running in a namespace that is monitored by the webhook by default.

| Name | Purpose  | Applied Manipulation  | Modified Manifest Field | Opt-In Annotation |
|--|--|--|--|--|
| Container Registry Rewrite | Replace container registry hosts with another host (e.g., for private container registries).| Rewrite container registry host in `image` field.| Rewrite registry hosts in `.spec.containers[*].image` | `rt-cfg.kyma-project.io/alter-img-registry: "true"`|
| Image Pull Secret Injection | The webhook ensures that the Secret resource exists in the namespace and adds a pull-secret entry to the manifest if the registry requires user credentials.| Add Secret reference to the `imagePullSecrets` field. | Append array `.spec.imagePullSecrets[]` with entry `registry-credentials` | `rt-cfg.kyma-project.io/add-img-pull-secret: "true"`|
| FIPS Mode Enablement| The webhook sets environment variables in the Pod to enable FIPS mode. | Add environment variables `KYMA_FIPS_MODE_ENABLED` and `FIPS_MODE_ENABLED`. | Append key-value array `.spec.containers[*].env[]` with `KYMA_FIPS_MODE_ENABLED=true` and `FIPS_MODE_ENABLED=true`   | `rt-cfg.kyma-project.io/set-fips-mode: "true"`     |
| Mount Cluster Trust Bundle Volume | Mount a certificate (stored as `ClusterTrustBundle`) as a projected volume into the container under the path `/etc/ssl/certs` (includes init-containers).| Mount a projected `volume` from `ClusterTrustBundle` to each container in the Pod under path `/etc/ssl/certs`. | 1. Add projected volume `rt-bootstrapper-certs` to `.spec.volumes[]`<br/>2. Mount this volume into each container under the mount path `/etc/ssl/certs` by extending the array `.spec.containers[*].volumeMounts` | `rt-cfg.kyma-project.io/add-cluster-trust-bundle: "true"` |

> [!NOTE]
> Once manipulated by the webhook, the Pod is annotated with `rt-bootstrapper.kyma-project.io/defaulted: "true"`.

### Example

This is an example of a Pod manifest before being intercepted by the Runtime Bootstrapper webhook. The annotations enable the webhook to:

1. Manipulate the image registry.
2. Add a pull secret (if needed).
3. Mount the `ClusterTrustBundle` as a projected volume.
4. Enable the FIPS mode.


```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: pause-test1
  labels:
    app: pause-test1
spec:
  replicas: 1
  selector:
    matchLabels:
      app: pause-test1
  template:
    metadata:
      annotations:
        rt-cfg.kyma-project.io/alter-img-registry: "true"
        rt-cfg.kyma-project.io/add-img-pull-secret: "true"
        rt-cfg.kyma-project.io/add-cluster-trust-bundle: "true"
        rt-cfg.kyma-project.io/set-fips-mode: "true"
      labels:
        app: pause-test1
    spec:
      containers:
      - name: pause
        image: replace.me/kyma-project/rt-bootstrapper/pause:e2e
```

This is the  Pod manifest after being processed by the webhook:

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: pause-test1
  labels:
    app: pause-test1
  annotations:
    rt-bootstrapper.kyma-project.io/defaulted: "true"
    rt-cfg.kyma-project.io/add-cluster-trust-bundle: "true"
    rt-cfg.kyma-project.io/add-img-pull-secret: "true"
    rt-cfg.kyma-project.io/alter-img-registry: "true"
    rt-cfg.kyma-project.io/set-fips-mode: "true"
spec:
  containers:
  - env:
    - name: KYMA_FIPS_MODE_ENABLED                             # FIPS mode enabled
      value: "true"
    - name: FIPS_MODE_ENABLED                                  # FIPS mode enabled (legacy)
      value: "true"
    image: ghcr.io/kyma-project/rt-bootstrapper/pause:e2e      # Registry host rewritten
    name: pause
    volumeMounts:                                              # ClusterTrustBundle as volume mounted
    - mountPath: /etc/ssl/certs
      name: rt-bootstrapper-certs
      readOnly: true
  imagePullSecrets:                                            # image-pull secret injected
  - name: registry-credentials
  volumes:
  - name: rt-bootstrapper-certs
    projected:
      defaultMode: 420
      sources:
      - clusterTrustBundle:
          name: rt-bootstrapper-k3d.test:ctb:1
          path: kube-apiserver-serving.pem
```

## High Level Flow

![High Level Flow](./assets/flow-manipulating-webhook.drawio.svg)

1. **Runtime Provisioning Initiation**:  
   Kyma Environment Broker (KEB) creates a Runtime custom resource (CR), which represents a Kyma runtime instance.

2. **Runtime CR Monitoring**:  
   Kyma Infrastructure Manager (KIM) continuously monitors changes to Runtime CRs.

3. **Kyma Runtime Provisioning**:  
   When a new Runtime CR is created, KIM provisions a new Kyma runtime based on a Gardener Cluster.

4. **Webhook Installation**:  
   Once the Kyma runtime is ready, KIM automatically installs the Runtime Bootstrapper webhook.

5. **Runtime CR Readiness**:  
   After the webhook is operational, KIM marks the Runtime CR as `Ready`.

6. **Runtime CR Status Monitoring**:  
   KEB monitors the status changes of Runtime CRs.

7. **Kyma Installation Initiation**:  
   After the Runtime is ready, KEB creates a Kyma CR, which represents a Kyma installation in the runtime.

8. **Kyma CR Monitoring**:  
   Kyma Lifecycle Manager (KLM) monitors the Kyma CR and reacts to newly created entities.

9. **Kyma Module Deployment**:  
   KLM begins deploying Kyma modules using the Kubernetes API server.

10. **Webhook Interception**:  
    The API server invokes the manipulating webhooks to intercept deployment requests.

11. **Request Deployment**:  
    The intercepted and manipulated requests are deployed on Kyma runtime.

12. **Kyma CR Readiness**:  
    Once all Kyma modules are successfully installed, KLM marks the Kyma CR as `Ready`.


## Resource Synchronization

A dedicated controller loop handles the synchronization of shared resources (for example, pull secret, `ClusterTrustBundle`, or webhook configuration). For details, see [the resource synchronizer documentation](./resource-synchronisation.md).

## Related Information
* [Architectural decision](./architectural-decisions.md)

