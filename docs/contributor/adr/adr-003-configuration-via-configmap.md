# ADR-003 – Configuration Using ConfigMap, No CRD

**Status:** Accepted

**Context:** Operators typically use custom resources to manage their configuration. This would require a CustomResourceDefinition (CRD) and a controller to reconcile it.

**Decision:** Configuration is stored in a standard ConfigMap managed by Kyma Infrastructure Manager (KIM). There is no CRD.

**Rationale:** A CRD would introduce a bootstrapping problem: the webhook cannot serve admission requests until it is running, but deploying the CRD requires the API server to be functional and the operator to have started. Using a ConfigMap removes this circularity and keeps the component operationally simpler.

**Consequences:** Configuration schema changes are not validated by the Kubernetes API server at write time. Validation is performed by `NewConfig()` at read time (inside the webhook process).
