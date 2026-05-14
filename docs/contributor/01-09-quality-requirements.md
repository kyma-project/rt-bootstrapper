# Quality Requirements

## Quality Tree

```
Quality
├── Reliability
│   ├── QS-1: Non-blocking webhook (API server availability)
│   └── QS-2: Idempotent manipulations
├── Correctness
│   ├── QS-3: Deterministic opt-in evaluation
│   └── QS-4: Configuration validation at read time
├── Security
│   ├── QS-5: TLS 1.3 minimum, HTTP/2 disabled by default
│   └── QS-6: FIPS-140-compliant binary
├── Operability
│   ├── QS-7: Automatic certificate rotation (no manual intervention)
│   └── QS-8: High scheduling priority (not displaced by user workloads)
└── Maintainability
    ├── QS-9: Extensible PodDefaulter pattern (new features without changing the dispatch loop)
    └── QS-10: Configuration-driven feature enablement (no redeployment for feature set changes)
```

## Quality Scenarios

| ID    | Quality Attribute | Stimulus                                                                                                   | Expected System Response                                                                                                                                                                             |
|-------|-------------------|------------------------------------------------------------------------------------------------------------|------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| QS-1  | Reliability       | The webhook Pod is restarted or temporarily unavailable.                                                   | The Kubernetes API server continues to admit Pod creation requests unmodified (`failurePolicy: Ignore`). Pods created during the outage are not automatically remediated; manual intervention is required to detect and restart them.             |
| QS-2  | Reliability       | The webhook is called twice for the same Pod (for example, due to a retry).                                | Each manipulation check is idempotent: no duplicate pull secrets, no double-mounted volumes, no duplicate env vars.                                                                                  |
| QS-3  | Correctness       | A Pod is created in a namespace listed in `namespaceFeatures` with no annotations on the namespace or Pod. | All features listed for that namespace in `namespaceFeatures` are applied to the Pod.                                                                                                                |
| QS-4  | Correctness       | KIM writes an invalid ConfigMap (unknown feature key in `namespaceFeatures`).                              | `NewConfig` returns an error; the webhook returns an admission error for the affected Pod rather than silently applying partial configuration.                                                       |
| QS-5  | Security          | A TLS client attempts to connect to the webhook server using TLS 1.2.                                      | The connection is rejected; TLS 1.3 is the minimum. HTTP/2 is not negotiated unless `--enable-http2` is explicitly set.                                                                              |
| QS-6  | Security          | The binary is inspected for FIPS-140 compliance.                                                           | The binary was compiled with `GOFIPS140=v1.0.0` and links the FIPS-140-validated Go crypto module.                                                                                                   |
| QS-7  | Operability       | cert-manager writes a renewed TLS certificate to the cert directory.                                       | Within seconds, `certwatcher` detects the new files, reloads the certificate, and patches the `MutatingWebhookConfiguration.caBundle` without any manual intervention or process restart.            |
| QS-8  | Operability       | The cluster is under heavy load and the scheduler must evict Pods to free resources.                       | The RT Bootstrapper Pod is not evicted because its `PriorityClass` value (2 100 000) is higher than standard user workloads.                                                                         |
| QS-9  | Maintainability   | A new manipulation is required (for example, inject a sidecar).                                            | A new `BuildXxx() PodDefaulter` constructor is added in `internal/webhook/v1/` and registered in `SetupPodWebhookWithManager`. No changes to the annotation evaluation or dispatch logic are needed. |
| QS-10 | Maintainability   | A feature must be enabled for an additional namespace.                                                     | KIM updates the `namespaceFeatures` entry in the ConfigMap. The change takes effect on the next Pod creation with no redeployment of Runtime Bootstrapper.                                           |
