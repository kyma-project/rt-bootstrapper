## 1. Update CTB Eligibility Helpers

- [x] 1.1 Add `CTBHashPresent(annotations map[string]string) bool` helper in `pkg/api/v1/ctb_values.go` that returns true when `rt-bootstrapper.kyma-project.io/ctb-hash` key exists in the annotations map
- [x] 1.2 Update `CTBRestartEnabled(annotations map[string]string) bool` in `pkg/api/v1/ctb_values.go` to return true when EITHER `add-cluster-trust-bundle` is `"true"` OR `ctb-hash` annotation is present
- [x] 1.3 Add unit tests for `CTBHashPresent` in `pkg/api/v1/ctb_values_test.go`: test with ctb-hash present, absent, and nil annotations
- [x] 1.4 Update unit tests for `CTBRestartEnabled` in `pkg/api/v1/ctb_values_test.go`: add cases for pod with ctb-hash but no add-cluster-trust-bundle annotation, pod with both, pod with neither

## 2. Update Restarter Tests

- [x] 2.1 Add test case in `internal/ctb/restarter_test.go`: pod with `ctb-hash` but without `add-cluster-trust-bundle` annotation and stale hash — verify pod is deleted
- [x] 2.2 Add test case in `internal/ctb/restarter_test.go`: pod with `ctb-hash` matching desired hash — verify pod is NOT deleted
- [x] 2.3 Add test case in `internal/ctb/restarter_test.go`: pod with `add-cluster-trust-bundle: "true"` but no `ctb-hash` — verify pod is deleted (treated as stale)
- [x] 2.4 Add test case in `internal/ctb/restarter_test.go`: orphan pod with stale `ctb-hash` but no owner references — verify pod is NOT deleted and warning is logged
- [x] 2.5 Add test case in `internal/ctb/restarter_test.go`: pod with no CTB-related annotations — verify pod is NOT deleted

## 3. Verify End-to-End Behavior

- [x] 3.1 Run existing unit tests (`go test ./pkg/api/v1/... ./internal/ctb/...`) and confirm all pass
- [x] 3.2 Run full test suite (`make test`) and confirm no regressions
