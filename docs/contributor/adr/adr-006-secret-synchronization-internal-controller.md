# ADR-006 – Secret Synchronization as an Internal Controller

**Status:** Accepted

**Context:** Pull Secrets are namespace-scoped; the same credentials must be available in every namespace. This could be handled by Kyma Infrastructure Manager (KIM) or an external sync tool.

**Decision:** A `SecretReconciler` inside the Runtime Bootstrapper binary watches and mirrors the master pull Secret to all namespaces.

**Rationale:** Keeping the Secret synchronization inside the same component makes Runtime Bootstrapper self-contained. KIM does not need to handle per-namespace secret management, and no additional tooling (for example, Reflector or external-secrets) is required.

**Consequences:** The controller performs a full namespace list on every master-Secret update. In clusters with thousands of namespaces this is a broad LIST operation; acceptable given that pull Secret updates are rare.
