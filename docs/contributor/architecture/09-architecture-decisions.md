# Architecture Decisions

## ADR-001 – Intercept Only Pods, Not Higher-Level Resources

**Status:** Accepted

**Context:** Kyma modules are deployed as Deployments, DaemonSets, and StatefulSets managed by KLM. Webhooks could intercept these resources as well.

**Decision:** The webhook intercepts only `Pod` resources (verb: `create`).

**Rationale:** KLM regularly reconciles and re-applies the resources it owns. If the webhook mutated a Deployment, KLM would overwrite the mutation on its next reconciliation cycle, creating an infinite fight. Pods are an exception: KLM does not create bare Pods directly, so mutations at the Pod level survive.

**Consequences:** The Pod spec can diverge from the template defined in its owning Deployment or StatefulSet, which may confuse engineers who read the parent resource without knowing about the webhook.

---

## ADR-002 – failurePolicy: Ignore (Non-Blocking Webhook)

**Status:** Accepted

**Context:** A mutating webhook with `failurePolicy: Fail` blocks all Pod creation when the webhook is unavailable.

**Decision:** The `MutatingWebhookConfiguration` is configured with `failurePolicy: Ignore`.

**Rationale:** Runtime Bootstrapper must not become a single point of failure for the Kubernetes API server. Losing the webhook temporarily is preferable to losing the ability to schedule any Pod.

**Consequences:** Pods created while the webhook is down will not be manipulated. An external housekeeping process (outside this repository) is responsible for detecting and restarting such Pods.

---

## ADR-003 – Configuration via ConfigMap, No CRD

**Status:** Accepted

**Context:** Operators typically use Custom Resources to manage their configuration. This would require a CRD and a controller to reconcile it.

**Decision:** Configuration is stored in a standard `ConfigMap` managed by KIM. There is no CRD.

**Rationale:** A CRD would introduce a bootstrapping problem: the webhook cannot serve admission requests until it is running, but deploying the CRD requires the API server to be functional and the operator to have started. Using a ConfigMap removes this circularity and keeps the component operationally simpler.

**Consequences:** Configuration schema changes are not validated by the Kubernetes API server at write time. Validation is performed by `NewConfig()` at read time (inside the webhook process).

---

## ADR-004 – Config Re-read on Every Webhook Invocation

**Status:** Accepted

**Context:** Caching the ConfigMap in memory is the default approach for performance.

**Decision:** `GetConfig` is called on every Pod admission request, fetching the ConfigMap from the API server every time.

**Rationale:** Configuration changes (for example, adding a new namespace to `namespaceFeatures`) must take effect immediately without restarting the webhook. The additional latency of one API server GET per admission request is acceptable given that the API server response is served from etcd cache.

**Consequences:** Each webhook invocation performs one additional API server read. Under very high Pod creation rates this could be noticeable; at typical Kyma workload rates it is negligible.

---

## ADR-005 – Self-Managed caBundle via certwatcher Callback

**Status:** Accepted

**Context:** cert-manager's `cainjector` can patch the `caBundle` field automatically via annotation. Alternatively, the webhook can manage it itself.

**Decision:** The webhook uses `certwatcher` to detect certificate file changes and patches the `MutatingWebhookConfiguration.caBundle` directly from within the process.

**Rationale:** This avoids a dependency on `cert-manager`'s `cainjector` component being present and healthy. The webhook becomes self-sufficient for its own certificate lifecycle, which simplifies the dependency chain in production environments where cert-manager configurations may differ.

**Consequences:** The webhook process requires RBAC permission to `get` and `patch` `MutatingWebhookConfiguration` resources.

---

## ADR-006 – Secret Synchronization as an Internal Controller

**Status:** Accepted

**Context:** Pull secrets are namespace-scoped; the same credentials must be available in every namespace. This could be handled by KIM or an external sync tool.

**Decision:** A `SecretReconciler` inside the Runtime Bootstrapper binary watches and mirrors the master pull secret to all namespaces.

**Rationale:** Keeping the secret synchronization inside the same component makes Runtime Bootstrapper self-contained. KIM does not need to handle per-namespace secret management, and no additional tooling (for example, Reflector or external-secrets) is required.

**Consequences:** The controller performs a full namespace list on every master-secret update. In clusters with thousands of namespaces this is a broad LIST operation; acceptable given that pull secret updates are rare.
