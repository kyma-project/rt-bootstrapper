# Runtime Bootstrapper

This folder contains documentation for the Runtime Bootstrapper (RT Bootstrapper). It is organized into two sections:

- [contributor/](contributor/) - Documentation for developers and contributors, including architecture details.
  Technical documentation for contributors and developers working on Runtime Bootstrapper. It covers the system architecture, design decisions, and supporting reference material needed to understand, extend, and maintain the codebase.
  - [Pod Manipulations](contributor/02-01-pod-manipulations.md)
  - [Architectural Decisions](contributor/adr/)
    - [ADR-001 – Intercept Only Pods, Not Higher-Level Resources](contributor/adr/adr-001-intercept-only-pods.md)
    - [ADR-002 – failurePolicy: Ignore (Non-Blocking Webhook)](contributor/adr/adr-002-failure-policy-ignore.md)
    - [ADR-003 – Configuration Using ConfigMap, No CRD](contributor/adr/adr-003-configuration-via-configmap.md)
    - [ADR-004 – Config Re-Read on Every Webhook Invocation](contributor/adr/adr-004-config-re-read-on-every-invocation.md)
    - [ADR-005 – Self-Managed caBundle Using certwatcher Callback](contributor/adr/adr-005-self-managed-cabundle.md)
    - [ADR-006 – Secret Synchronization as an Internal Controller](contributor/adr/adr-006-secret-synchronization-internal-controller.md)
    - [ADR-007 – Direct Runtime Configuration Synchronization](contributor/adr/adr-007-direct-runtime-configuration-sync.md)
  - [Architecture: Requirements and Goals](contributor/01-01-requirements-and-goals.md)
  - [Architecture Constraints](contributor/01-02-architecture-constraints.md)
  - [System Scope and Context](contributor/01-03-context-and-scope.md)
  - [Solution Strategy](contributor/01-04-solution-strategy.md)
  - [Building Block View](contributor/01-05-building-block-view.md)
  - [Runtime View](contributor/01-06-runtime-view.md)
  - [Deployment View](contributor/01-07-deployment-view.md)
  - [Cross-Cutting Concepts](contributor/01-08-crosscutting-concepts.md)
  - [Quality Requirements](contributor/01-09-quality-requirements.md)
  - [Risks and Technical Debt](contributor/01-10-risks-and-technical-debt.md)
  - [Configuration Synchronization Using Controller Loop](contributor/01-11-resource-synchronization.md)
  - [Glossary](contributor/01-12-glossary.md)
  - [Pod Manipulations](contributor/02-01-pod-manipulations.md)
  - [Testing Strategy](contributor/02-02-testing-strategy.md)
