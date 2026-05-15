# ADR-004 – Config Re-Read on Every Webhook Invocation

**Status:** Accepted

**Context:** Caching the ConfigMap in memory is the default approach for performance.

**Decision:** `GetConfig` is called on every Pod admission request, fetching the ConfigMap from the API server every time.

**Rationale:** Configuration changes (for example, adding a new namespace to `namespaceFeatures`) must take effect immediately without restarting the webhook. The additional latency of one API server GET per admission request is acceptable given that the API server response is served from etcd cache.

**Consequences:** Each webhook invocation performs one additional API server read. Under very high Pod creation rates this could be noticeable; at typical Kyma workload rates, it is negligible.
