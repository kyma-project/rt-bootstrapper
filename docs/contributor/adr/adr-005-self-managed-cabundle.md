# ADR-005 – Self-Managed caBundle Using certwatcher Callback

**Status:** Accepted

**Context:** cert-manager's `cainjector` can patch the `caBundle` field automatically through annotation. Alternatively, the webhook can manage it itself.

**Decision:** The webhook uses `certwatcher` to detect certificate file changes and patches the `MutatingWebhookConfiguration.caBundle` directly from within the process.

**Rationale:** This avoids a dependency on cert-manager's `cainjector` component being present and healthy. The webhook becomes self-sufficient for its own certificate lifecycle, which simplifies the dependency chain in production environments where cert-manager configurations may differ.

**Consequences:** The webhook process requires RBAC permission to GET and PATCH `MutatingWebhookConfiguration` resources.
