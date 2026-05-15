# ADR-002 – failurePolicy: Ignore (Non-Blocking Webhook)

**Status:** Accepted

**Context:** A mutating webhook with `failurePolicy: Fail` blocks all Pod creation when the webhook is unavailable.

**Decision:** The `MutatingWebhookConfiguration` is configured with `failurePolicy: Ignore`.

**Rationale:** Runtime Bootstrapper must not become a single point of failure for the Kubernetes API server. Losing the webhook temporarily is preferable to losing the ability to schedule any Pod.

**Consequences:** Pods created while the webhook is down are not manipulated. An external housekeeping process (outside this repository) is responsible for detecting and restarting such Pods.
