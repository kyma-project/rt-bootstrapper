## 1. Webhook: Stamp ctb-hash on all CTB-opted-in pods (including orphans)

- [x] 1.1 In `BuildDefaulterAddClusterTrustBundle` (`internal/webhook/v1/pod_defaulters.go`), move the hash-stamping logic out of the `CTBRestartEnabled(p.Annotations)` guard so it applies to every pod that receives the CTB volume (still gated on non-empty hash from hash holder)
- [x] 1.2 Remove the orphan guard (`len(p.OwnerReferences) == 0`) from the hash-stamping logic in the webhook — orphan pods SHALL receive the `ctb-hash` annotation
- [x] 1.3 Add unit test: pod opted in via namespace defaults (no pod-level `add-cluster-trust-bundle` annotation) receives `ctb-hash` stamp
- [x] 1.4 Add unit test: pod opted in via namespace annotation (no pod-level annotation) receives `ctb-hash` stamp
- [x] 1.5 Add unit test: orphan pod (no owner references) DOES receive `ctb-hash` stamp when CTB volume is mounted

## 2. Restarter: Treat missing hash as stale

- [x] 2.1 Verify that `restartStalePodsInNamespace` (`internal/ctb/restarter.go`) correctly handles pods where `CTBRestartEnabled` returns true but `ctb-hash` annotation is absent — the empty string comparison `"" == desiredHash` already returns false, so these pods should be deleted. Confirm with a test (no code change expected).

## 3. Restarter tests

- [x] 3.1 Add test: pod with `add-cluster-trust-bundle: "true"` but no `ctb-hash` annotation AND with owner references is deleted (treated as stale)
- [x] 3.2 Add test: pod with `ctb-hash: "old-hash"` but WITHOUT `add-cluster-trust-bundle` annotation is NOT deleted (not eligible)
- [x] 3.3 Add test: pod with `ctb-hash` matching desired hash is NOT deleted
- [x] 3.4 Add test: orphan pod with `add-cluster-trust-bundle: "true"` but no owner references is NOT deleted, and a warning is logged
- [x] 3.5 Verify existing tests still pass after the changes

## 4. Helper functions (if needed)

- [x] 4.1 Evaluate whether a new helper `CTBRestartEligible(annotations)` in `pkg/api/v1/ctb_values.go` is cleaner than inline logic in the restarter — if so, add it with tests in `ctb_values_test.go`

## 5. Validation

- [x] 5.1 Run full test suite (`go test ./...`) and verify all tests pass
- [x] 5.2 Run linter (`make lint` or `golangci-lint run`) and fix any issues
