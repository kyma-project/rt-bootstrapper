# Runtime Bootstrapper — KPI Framework

> **Version:** 1.0  
> **Date:** 2026-11-11  
> **Stakeholders:** Team, Varban, Artur  
> **Review cadence:** Quarterly

---

## Purpose

This document defines the Key Performance Indicators (KPIs) for the Runtime Bootstrapper (RT Bootstrapper), the Kyma component that automatically configures Kyma runtime instances in SAP BTP landscapes by intercepting and mutating Pod specifications at admission time. It covers:

- What each KPI measures and how it is collected
- Why it was chosen and what business value it reflects
- Thresholds (warning and critical) and the action plan for each breach

The KPIs are designed to surface technical performance, reliability, scalability, and product health in a form that is meaningful both to the engineering team and to management stakeholders.

---

## Table of Contents

1. [KPI Overview](#1-kpi-overview)
2. [Reliability KPIs](#2-reliability-kpis)
   - [2.1 Webhook Availability](#21-webhook-availability)
   - [2.2 Pod Mutation Success Rate](#22-pod-mutation-success-rate)
   - [2.3 Secret Sync Drift](#23-secret-sync-drift)
3. [Performance KPIs](#3-performance-kpis)
   - [3.1 Webhook Admission Latency (p99)](#31-webhook-admission-latency-p99)
   - [3.2 Webhook Error Rate](#32-webhook-error-rate)
   - [3.3 Secret Sync Reconciliation Latency](#33-secret-sync-reconciliation-latency)
4. [Scalability KPIs](#4-scalability-kpis)
   - [4.1 Admission Request Throughput Capacity](#41-admission-request-throughput-capacity)
5. [Product Health KPIs](#5-product-health-kpis)
   - [5.1 Component Availability (Pod Uptime)](#51-component-availability-pod-uptime)
   - [5.2 Certificate Rotation Lead Time](#52-certificate-rotation-lead-time)
   - [5.3 Configuration Propagation Correctness](#53-configuration-propagation-correctness)
6. [Code Quality and Maintenance KPIs](#6-code-quality-and-maintenance-kpis)
   - [6.1 Test Coverage](#61-test-coverage)
   - [6.2 Critical Open Bug Age](#62-critical-open-bug-age)
   - [6.3 Mean Time to Recovery (MTTR)](#63-mean-time-to-recovery-mttr)
7. [Measurement Infrastructure](#7-measurement-infrastructure)
8. [Review Process and Stakeholder Alignment](#8-review-process-and-stakeholder-alignment)

---

## 1. KPI Overview

| # | KPI | Category | Target | Warning | Critical |
|---|-----|----------|--------|---------|----------|
| 2.1 | Webhook Availability | Reliability | ≥ 99.9 % | < 99.9 % | < 99.5 % |
| 2.2 | Pod Mutation Success Rate | Reliability | 100 % | < 100 % | < 98 % |
| 2.3 | Secret Sync Drift | Reliability | 0 namespaces out of sync | 1–3 namespaces > 2 min | > 3 namespaces or any > 5 min |
| 3.1 | Webhook Admission Latency p99 | Performance | ≤ 200 ms | > 200 ms | > 500 ms |
| 3.2 | Webhook Error Rate | Performance | ≤ 0.1 % | > 0.1 % | > 1.0 % |
| 3.3 | Secret Sync Reconciliation Latency p95 | Performance | ≤ 5 s | > 5 s | > 15 s |
| 4.1 | Admission Request Throughput Capacity | Scalability | ≥ baseline | < 80 % baseline | < 60 % baseline |
| 5.1 | Component Availability (Pod Uptime) | Product Health | ≥ 99.9 % | < 99.9 % | < 99.5 % |
| 5.2 | Certificate Rotation Lead Time | Product Health | ≤ 30 s | > 30 s | > 120 s |
| 5.3 | Configuration Propagation Correctness | Product Health | 100 % correct at next Pod creation | stale reads > 2 min | systematic stale config |
| 7.1 | Test Coverage | Quality | ≥ 80 % | < 80 % | < 70 % |
| 7.2 | Critical Open Bug Age | Quality | ≤ 5 business days | > 5 days | > 10 days |
| 7.3 | MTTR | Quality | ≤ 4 h | > 4 h | > 8 h |

---

## 2. Reliability KPIs

### 2.1 Webhook Availability

**Definition:** Percentage of time during which the RT Bootstrapper webhook is reachable by the Kubernetes API server and returns a valid admission response (permit or denial), measured over a rolling 30-day window. A "failed" request is one that times out or returns a 5xx HTTP response at the webhook level — not a deliberate admission denial due to invalid configuration.

**How it is measured:**
- Source: Prometheus metric `controller_runtime_webhook_requests_total{code!~"2.."}` vs. `controller_runtime_webhook_requests_total`.
- Aggregated over a rolling 30-day window.
- Alerting via a multi-window burn-rate approach (1 h and 6 h error rates) to detect short bursts and slow degradation.

**Why it was chosen:**
The webhook uses `failurePolicy: Ignore`, which means any period of unresponsiveness causes Pods to be admitted *without* the required landscape-specific mutations — missing image-pull Secrets, missing CA certificates, and missing FIPS flags. This is a silent degradation: workloads appear healthy but are running in a misconfigured state. Webhook availability is therefore the foundation on which all correctness guarantees rest and is the single most critical operational signal for the component.

**Thresholds and action plan:**

**Warning** — < 99.9 % in a 30-day window:
1. Identify the affected time window using the burn-rate alert metadata.
2. Check Pod events and logs for OOM kills, CrashLoops, or certificate-related errors.
3. Review recent deployments or ConfigMap changes.
4. Adjust resource limits or certificate configuration if needed.
5. Notify the team at the next standup.

**Critical** — < 99.5 % in a 30-day window, or a sustained window with 0 Ready replicas:
1. Declare incident.
2. Page on-call engineer.
3. Use Kubernetes audit logs to identify all Pods created during the degraded window.
4. Re-deploy those Pods to apply the missing mutations.
5. Roll back the last deployment or configuration change if correlated.
6. Post incident root-cause analysis.

---

### 2.2 Pod Mutation Success Rate

**Definition:** Percentage of admitted Pods that are stamped with the `rt-bootstrapper.kyma-project.io/modified: "true"` annotation among all Pods that were expected to be mutated — that is, opted in via `namespaceFeatures` configuration, or carrying a namespace-level or Pod-level opt-in annotation. Measured over a rolling 24-hour window.

**How it is measured:**
- Source: A Prometheus counter instrumented inside the webhook handler, labelled by outcome (`modified` or `skipped`), combined with cluster audit log analysis.
- Alternatively: periodic comparison of `kubectl get pods -A -l rt-bootstrapper.kyma-project.io/modified=true` against the expected opted-in Pod set.

**Why it was chosen:**
This is the primary correctness KPI — it directly answers "Is the RT Bootstrapper actually doing its job?" A gap between opted-in Pods and modified Pods means workloads may be pulling images from the wrong registry, missing CA certificates needed for TLS communication with BTP backend services, or running without FIPS environment flags. Silent gaps are particularly dangerous because the workload may appear healthy while violating compliance and operational requirements. This KPI validates quality scenarios QS-2 and QS-3.

**Thresholds and action plan:**

**Warning** — any Pod created in an opted-in namespace is found without the `modified` annotation:
1. Cross-reference with K-01 (webhook availability) degradation windows — the Pod may have been created during an outage.
2. If the webhook was healthy, investigate whether the namespace annotation or `namespaceFeatures` entry is correctly configured.
3. Restart affected Pods if safe to do so.

**Critical** — mutation success rate < 98 %, or a sustained gap across an entire namespace:
1. Declare incident.
2. Halt new Kyma module rollouts until the root cause is identified.
3. Identify all un-modified Pods across the landscape using audit logs.
4. Perform a controlled rolling restart of Pods in affected namespaces.
5. Escalate to KIM team if the cause is a ConfigMap misconfiguration.
6. Communicate impact to operators and module teams.

---

### 2.3 Secret Sync Drift

**Definition:** Number of namespaces in the cluster where the `registry-credentials` Secret is absent or has content different from the master copy in `kyma-system`, at any point in time. The healthy state is zero: all namespaces have an up-to-date copy.

**How it is measured:**
- Source: A Prometheus gauge emitted by the Secret sync controller after each reconciliation cycle, recording the count of out-of-sync namespaces.
- Alternatively derived by a periodic scan comparing Secret checksums across all namespaces.

**Why it was chosen:**
The Secret sync controller mirrors image-pull credentials to every namespace so that Pods anywhere in the cluster can pull from private registries. A namespace without the Secret causes `ImagePullBackOff` for every workload deployed there — a hard failure, not a degraded-mode failure. This affects both Kyma module Pods (managed by KLM) and customer workloads. Drift can occur after a controller crash, transient API error, or when a new namespace is created faster than the reconciler reacts.

**Thresholds and action plan:**

**Warning** — 1–3 namespaces out of sync:
1. Verify the Secret sync controller Pod is running and reconciling.
2. Check controller logs for API server throttling (429 errors) or other transient errors.
3. Trigger a manual reconciliation by annotating or touching the master Secret in `kyma-system`.

**Critical** — more than 3 namespaces out of sync, or any namespace persistently out of sync:
1. Declare incident.
2. Identify workloads in affected namespaces currently in `ImagePullBackOff`.
3. Manually push the Secret to affected namespaces as immediate remediation.
4. Investigate root cause: controller crash, network partition, or API server outage.
5. Review controller resource limits to prevent recurrence.

---

## 3. Performance KPIs

### 3.1 Webhook Admission Latency (p99)

**Definition:** The 99th-percentile round-trip latency of a single webhook admission call, measured from the moment the Kubernetes API server sends the `AdmissionReview` request to the moment it receives the response. Measured over a rolling 5-minute window.

**How it is measured:**
- Source: Prometheus histogram `controller_runtime_webhook_latency_seconds`.
- Query: `histogram_quantile(0.99, rate(controller_runtime_webhook_latency_seconds_bucket[5m]))`.
- Reported as a real-time panel on the team dashboard with a 30-day trend.

**Why it was chosen:**
The webhook is called synchronously on every Pod creation. High latency directly delays Pod scheduling, which cascades into slow Kyma module rollouts, slow provisioning of new Kyma runtimes, and degraded CI/CD pipeline throughput for module teams. The architecture reads the ConfigMap on every invocation (ADR-004); latency spikes therefore also reveal API server pressure or network degradation. The p99 is chosen over the average to surface worst-case scheduling delays that affect the tail of users and automated provisioning pipelines.

**Thresholds and action plan:**

**Warning** — p99 > 200 ms sustained:
1. Profile the webhook handler to identify whether the delay is in ConfigMap GET, image rewriting, or Pod spec manipulation.
2. Check for API server latency using `apiserver_request_duration_seconds`.
3. Confirm the webhook Pod is not CPU-throttled by comparing `container_cpu_throttled_seconds_total`.

**Critical** — p99 > 500 ms sustained:
1. Declare incident.
2. If ConfigMap GET latency is the bottleneck, evaluate switching to the informer-based in-process cache described in ADR-004.
3. If the bottleneck is queue saturation, consider scaling up the Deployment replicas.
4. Alert KIM team if the cluster's API server is the root cause.

---

### 3.2 Webhook Error Rate

**Definition:** Percentage of admission requests that result in a non-deliberate error response (HTTP 5xx or timeout), excluding intentional admission denials due to invalid ConfigMap content, which are correct behaviour as specified in QS-4. Measured over a rolling 1-hour window.

**How it is measured:**
- Source: `controller_runtime_webhook_requests_total{code=~"5.."}` and panic/crash events extracted from Pod logs.
- A breakdown by feature type (image registry, FIPS, trust bundle, landscape, pull secret) enables root-cause isolation.

**Why it was chosen:**
Even at low absolute rates, repeated errors in a specific feature path or namespace indicate a systematic coding defect or configuration problem. Because `failurePolicy: Ignore` masks errors at the cluster level, the webhook error rate is the only signal that directly surfaces bugs in the mutation logic. This KPI distinguishes operational noise from regressions.

**Thresholds and action plan:**

**Warning** — error rate > 0.1 %:
1. Check logs for panic stack traces or unhandled errors.
2. Identify the feature type or namespace triggering the errors using label breakdown.
3. Deploy a hotfix if the error is isolated to a single code path.

**Critical** — error rate > 1.0 %:
1. Declare incident.
2. Roll back to the previous stable release if errors are widespread and caused by a recent deployment.
3. File a priority bug with a reproduction case.
4. Conduct a post-incident review.
5. Add a targeted E2E test to cover the failure scenario before the next release.

---

### 3.3 Secret Sync Reconciliation Latency

**Definition:** Time elapsed from the moment a new namespace is created to the moment the `registry-credentials` Secret is present and correct in that namespace. Measured as p95 across all new namespace creation events per day.

**How it is measured:**
- Source: A `secret_sync_latency_seconds` histogram gauge emitted by the Secret sync controller, capturing the delta between the namespace creation timestamp and the Secret write completion timestamp.

**Why it was chosen:**
If a Kyma module or customer workload deploys a Pod immediately after namespace creation — a common pattern in automated provisioning — and the Secret has not yet been synced, the Pod fails with `ImagePullBackOff`. The `createNsPredicate` is designed to minimise this window, but controller queue depth, network latency, and API server load can extend it. This KPI validates the responsiveness of F-5 and the timeliness guarantee that new namespaces receive credentials before their first Pod is scheduled.

**Thresholds and action plan:**

**Warning** — p95 > 5 seconds:
1. Check controller queue depth and processing backlog.
2. Verify namespace creation events are delivered promptly to the controller (no watch lag).
3. Look for CPU throttling or API server rate limiting in controller logs.

**Critical** — p95 > 15 seconds:
1. Declare incident.
2. Increase controller resource limits.
3. Review whether the `SecretSyncInterval` (default: 1 minute) needs to be reduced as a stopgap.
4. Investigate whether a Namespace admission webhook that pre-populates the Secret is warranted for high-churn environments.

---

## 4. Scalability KPIs

### 4.1 Admission Request Throughput Capacity

**Definition:** Maximum sustainable Pod admission requests per second processed by the RT Bootstrapper webhook under a fixed load-test profile while meeting the latency (≤ 200 ms p99) and error-rate (≤ 0.1 %) targets defined in sections 3.1 and 3.2.

**How it is measured:**
- Source: Load-test results run against a reference cluster configuration on each release (e.g., using k6 or a Kubernetes Pod creation burst test).
- Production Prometheus rate metrics are collected as a traffic signal only and not used for lower-bound health alerts, since legitimate quiet periods would create false positives.
- Reported as: sustainable RPS at the defined load profile per release, tracked over time for regression.

**Why it was chosen:**
RT Bootstrapper sits on the critical admission path for every Pod creation in every Kyma runtime. As the number of Kyma module rollouts, customer workload deployments, and concurrent runtime provisioning events grows, the webhook must scale without becoming a bottleneck. A load-test-derived capacity metric catches performance regressions between releases before they affect production, especially regressions caused by changes to the per-invocation ConfigMap GET or the image-rewriting logic.

**Thresholds and action plan:**

**Warning** — capacity drops by > 20 % compared to the previous release under the same load profile:
1. Profile the regression: is the bottleneck CPU, memory, API server round-trips, or in-process computation?
2. Check whether recent changes to the ConfigMap GET path or Pod manipulation logic introduced extra allocations or synchronisation.
3. Confirm resource requests/limits and HPA settings are not artificially capping throughput.

**Critical** — capacity drops by > 40 % compared to the previous release, or the load test fails to complete:
1. Block the release.
2. Identify the commit(s) responsible for the regression using `git bisect` against the load-test results.
3. Fix and re-run the load test before re-staging the release.

---

## 5. Product Health KPIs

### 5.1 Component Availability (Pod Uptime)

**Definition:** Percentage of time the RT Bootstrapper Deployment has at least one Ready Pod, measured over a rolling 30-day window. This captures restarts, OOM kills, evictions, and cluster-level failures.

**How it is measured:**
- Source: `kube_deployment_status_replicas_available` for `rt-bootstrapper-controller-manager`, correlated with `kube_pod_container_status_restarts_total` to track restart frequency.

**Why it was chosen:**
Pod uptime is the operational health proxy watched by the on-call team and a prerequisite for webhook availability (K-01). While `failurePolicy: Ignore` prevents RT Bootstrapper downtime from blocking Pod creation entirely, any downtime causes silent mutation gaps (K-04) and FIPS compliance gaps (K-06). The high PriorityClass (value: 2,100,000) should prevent eviction under cluster load (QS-8); unexpected restarts therefore indicate resource limit misconfiguration, OOM conditions, or application bugs.

**Thresholds and action plan:**

**Warning** — < 99.9 % over 30 days, or more than 2 Pod restarts in a rolling window:
1. Review Pod restart history via `kubectl describe pod`.
2. Check for OOM kills (look for `OOMKilled` in `lastState`).
3. Adjust memory requests and limits if OOM kills are confirmed.
4. Verify the PriorityClass is applied correctly.

**Critical** — < 99.5 % over 30 days, or a sustained window with 0 Ready replicas:
1. Initiate incident response.
2. If OOM kills are persistent, increase memory limits as an immediate fix and file a resource sizing ticket.
3. If the PriorityClass has been removed or downgraded, restore it immediately.
4. Investigate whether a cluster control plane degradation is the root cause.

---

### 5.2 Certificate Rotation Lead Time

**Definition:** Time elapsed from the moment cert-manager writes a renewed TLS certificate to the certificate directory to the moment the RT Bootstrapper has reloaded the certificate *and* patched the `MutatingWebhookConfiguration.caBundle`. Measured per rotation event.

**How it is measured:**
- Source: Compare cert-manager certificate renewal event timestamp (Kubernetes Events) with the timestamp of the `caBundle` patch on `MutatingWebhookConfiguration` (Kubernetes audit logs or a dedicated metric emitted by the `certwatcher` callback).

**Why it was chosen:**
If the certificate rotation lag is too long, the Kubernetes API server may attempt to connect to the webhook using the new certificate while the `caBundle` still references the old public key, causing all admission requests to fail with TLS handshake errors. This would fully disable Pod mutations and, if `failurePolicy` were ever changed, could block all Pod creation. QS-7 mandates that rotation is fully automatic with no manual intervention; this KPI verifies that the `certwatcher`-based hot-reload mechanism meets that requirement under realistic conditions.

**Thresholds and action plan:**

**Warning** — lead time > 30 seconds:
1. Check `certwatcher` logs for file system notification delays.
2. Verify cert-manager is writing the certificate atomically and not in partial writes.
3. Confirm the `caBundle` patcher has the correct RBAC permissions (`MutatingWebhookConfiguration` update).

**Critical** — lead time > 120 seconds, or the API server is actively rejecting webhook calls due to certificate mismatch:
1. Manually patch the `caBundle` to immediately unblock admission.
2. File a bug against the `certwatcher` integration.
3. Review whether `inotify` limits or OS file watcher quotas are causing the notification delay.
4. Do not allow the certificate to expire; escalate to the infrastructure team if cert-manager is the root cause.

---

### 5.3 Configuration Propagation Correctness

**Definition:** Whether the ConfigMap content written by KIM is correctly applied to the next Pod creation after the write. Because the webhook reads the ConfigMap on every invocation (ADR-004), the propagation delay equals the time between the ConfigMap update and the next Pod creation; there is no artificial caching delay. This KPI measures *correctness* rather than speed: the goal is that the next Pod after a ConfigMap update receives mutations based on the updated config, not stale content.

**How it is measured:**
- Source: E2E test in CI: update the ConfigMap, immediately create a Pod, verify the mutation reflects the new config. Tracked as a pass/fail quality gate per release.
- Production: monitor webhook logs for repeated ConfigMap GET errors (403, 429, or network errors), which would cause stale config to be used.

**Why it was chosen:**
KIM uses ConfigMap updates to roll out new landscape configurations (new namespace features, registry overrides, FIPS enablement). If a stale or erroneous read causes the wrong mutations to be applied, Pods may be misconfigured in ways that are not immediately visible — wrong registry, wrong landscape identifier, or missing FIPS flags. ADR-004 explicitly traded per-invocation simplicity for immediate propagation; this KPI verifies that trade-off holds in practice.

**Thresholds and action plan:**

**Warning** — ConfigMap GET errors detected in webhook logs:
1. Check API server availability and RBAC permissions for the ConfigMap GET operation.
2. Look for 403 (permissions) or 429 (rate limiting) errors in webhook logs.
3. Verify the `rt-bootstrapper-config` ConfigMap exists and is valid.

**Critical** — systematic stale configuration causing incorrect mutations confirmed across multiple Pods:
1. Consider temporarily switching to the informer-based cache described in ADR-004 as an emergency mitigation.
2. Raise a priority bug.
3. Notify KIM team of affected landscapes and the scope of incorrectly configured Pods.
4. Perform a controlled rolling restart of affected Pods after the ConfigMap is restored to a correct state.

---

## 6. Code Quality and Maintenance KPIs

### 6.1 Test Coverage

**Definition:** Percentage of RT Bootstrapper Go code (excluding generated files and test files themselves) covered by unit and integration tests.

**How it is measured:**
- Source: `go test ./... -coverprofile cover.out` run in CI on every pull request.
- Coverage reported per package and as an aggregate.
- Special attention to: webhook mutation paths, idempotency guards, Secret sync controller, and certificate rotation callback.

**Why it was chosen:**
The RT Bootstrapper's correctness guarantees (idempotent mutations, correct annotation evaluation, valid TLS handling) are non-trivial and regression-prone. Low coverage in mutation paths, error recovery branches, or idempotency guards means defects ship silently. This KPI maintains the discipline needed to keep the component reliable as new features (new `PodDefaulter` types) are added.

**Thresholds and action plan:**

**Warning** — aggregate coverage drops below 80 %, or coverage for `internal/webhook/v1/` drops below 75 %:
1. Review recent PRs that reduced coverage.
2. Add missing tests before merging the next feature.
3. Prioritise coverage of: idempotency edge cases, error recovery in Secret sync, `certwatcher` callback logic, and annotation precedence evaluation.

**Critical** — aggregate coverage drops below 70 %:
1. Freeze new feature work until coverage is restored.
2. Identify and resolve the largest uncovered blocks.
3. Treat as a quality incident — add to the team's sprint as explicit tasks.

---

### 6.2 Critical Open Bug Age

**Definition:** Age of a bug labelled `priority/critical` or `priority/high` that remains open without a fix merged.

**How it is measured:**
- Source: GitHub Issues with the relevant priority labels in this repository.
- Measured as: age from issue creation to PR merge or issue close.
- Reported as median and maximum open age at the weekly team review.

**Why it was chosen:**
Critical bugs left open degrade the operational reliability and security posture of the component. Because RT Bootstrapper runs at cluster admission level, an unresolved critical bug can affect every Pod created in a Kyma runtime. Without an explicit KPI, high-priority bugs can accumulate behind feature work. This KPI enforces accountability and ensures the team maintains a healthy bug backlog.

**Thresholds and action plan:**

**Warning** — any critical/high bug open without an assignee or active fix in progress:
1. Assign the bug to a specific engineer at the next standup.
2. Break the fix into a minimally scoped PR to avoid stalling.

**Critical** — any critical bug stalled without progress:
1. Escalate to team lead.
2. Re-evaluate whether the priority label is accurate — if it truly is critical, it displaces sprint work.
3. Document the blocker explicitly if an external dependency is causing the delay.

---

### 6.3 Mean Time to Recovery (MTTR)

**Definition:** Average time from the point an incident is declared (or a critical alert fires) to the point the affected KPI returns within its normal threshold.

**How it is measured:**
- Source: Incident records (GitHub Issues tagged `incident` or an equivalent tracking system).
- Measured from the timestamp of the first alert or incident declaration to the timestamp of confirmed resolution.
- Reported as a rolling 90-day average.

**Why it was chosen:**
MTTR is the most direct measure of the team's ability to respond to production issues. Because RT Bootstrapper's incidents (webhook downtime, mutation gaps, FIPS non-compliance) silently affect workloads without immediately visible failures, a high MTTR amplifies the scope of every incident. It also drives investment decisions: a consistently high MTTR signals a need for better runbooks, on-call tooling, or more automated recovery mechanisms.

**Thresholds and action plan:**

**Warning** — average MTTR exceeds the target threshold:
1. Review the last 3 incidents: which step took the most time?
2. If diagnosis was slow: invest in better Grafana dashboards, alerting context, or runbooks for the specific KPI that was breached.
3. If remediation was slow: identify which manual steps (e.g., identify un-modified Pods, manually push Secrets) could be automated or pre-approved.

**Critical** — average MTTR significantly exceeds the target, or a single incident remains unresolved for an extended period:
1. Hold a dedicated retrospective.
2. Identify systemic gaps: missing runbook, insufficient alerting, unclear on-call responsibility, or external dependency.
3. Create follow-up tasks with owners and due dates.

---

## 7. Measurement Infrastructure

### Data Sources

| Source | Provides |
|--------|---------|
| Prometheus + `controller-runtime` default metrics | Webhook request counts, latency histograms, error rates, reconciliation counters |
| Prometheus + custom controller metrics | Secret sync latency, out-of-sync namespace gauge, mutation outcome counters |
| Kubernetes API (Pod labels, Deployment status) | Mutation success rate, component availability |
| Kubernetes Events and audit logs | Certificate rotation timestamps, ConfigMap update timestamps, Pod creation timestamps |
| CI/CD pipeline results | Test coverage, throughput capacity (load test), configuration propagation correctness |
| GitHub Issues | Bug age, incident count |
| Incident tracking | MTTR |

### Dashboard

A shared Grafana dashboard should be maintained with:

- One panel per KPI from the overview table (Section 1).
- Traffic-light colour coding matching the warning/critical thresholds.
- A 30-day historical trend view for reliability KPIs (2.1, 2.2, 5.1).
- A real-time view for performance KPIs (3.1, 3.2) over rolling 5-minute windows.
- An incident log panel showing open incidents and their current age.

### Alerting

Prometheus alerting rules should be defined for all KPIs measurable in real time (sections 2–6.1). Alert severity should map to the warning/critical levels defined per KPI. Alerts should fire to the team's primary notification channel; critical alerts additionally page the on-call engineer.

---

## 8. Review Process and Stakeholder Alignment

### Quarterly KPI Review

The team conducts a quarterly KPI review covering:

1. **Status review** — traffic-light status for all KPIs against their targets.
2. **Trend analysis** — which KPIs are improving, stable, or degrading over time?
3. **Action item follow-up** — were action items from the previous quarter completed?
4. **Stakeholder session** — Varban and Artur are invited to the quarterly review to:
   - Validate that the KPIs still reflect business objectives and regulatory requirements.
   - Raise new requirements or adjust thresholds based on customer feedback.
   - Review open incidents and their resolution quality.

### Feedback Loop

This document is a living specification. Changes to KPI definitions, thresholds, or action plans should be:

- Proposed as a pull request to this file.
- Reviewed by at least one team member and one stakeholder (Varban or Artur) before merging.
- Announced to the team in the next standup after merging.

### Continuous Improvement Commitment

The KPIs defined here support continuous improvement by:

- **Proactive issue identification:** Real-time alerting on warning thresholds catches degradation before it becomes critical.
- **Transparency across teams:** The shared dashboard and quarterly review ensure that reliability, correctness, and compliance are visible to all stakeholders, not just engineers.
- **Learning from incidents:** MTTR tracking and post-mortems feed directly into improvements to runbooks, alerting, and architecture.
- **Shared understanding:** KPI definitions, measurement methods, and action plans are documented here so any team member can interpret a metric consistently and act without ambiguity.

---

## Related Documents

- [Requirements and Goals](contributor/01-01-requirements-and-goals.md)
- [Quality Requirements](contributor/01-09-quality-requirements.md)
- [Risks and Technical Debt](contributor/01-10-risks-and-technical-debt.md)
- [ADR Index](contributor/adr/README.md) — ADR-001 through ADR-007 for architectural decisions that inform KPI thresholds
