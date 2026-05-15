# ADR-001 – Intercept Only Pods, Not Higher-Level Resources

**Status:** Accepted

**Context:** Kyma modules are deployed as Deployments, DaemonSets, and StatefulSets managed by Kyma Lifecycle Manager (KLM). Webhooks could intercept these resources as well.

**Decision:** The webhook intercepts only Pod resources (verb: `create`).

**Rationale:** KLM regularly reconciles and re-applies the resources it owns. If the webhook mutated a Deployment, KLM would overwrite the mutation on its next reconciliation cycle, creating an infinite fight. Pods are an exception: KLM does not create bare Pods directly, so mutations at the Pod level survive.

**Consequences:** The Pod spec can diverge from the template defined in its owning Deployment or StatefulSet, which may confuse engineers who read the parent resource without knowing about the webhook.
