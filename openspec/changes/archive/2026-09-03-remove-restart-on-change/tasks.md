# Tasks: Remove `restart-on-change` annotation value

## 1. Update API constants and helpers

- [ ] 1.1 Remove `CTBValueRestartOnChange` constant from `pkg/api/v1/ctb_values.go`
- [ ] 1.2 Update `CTBMountEnabled` to return `true` only for `"true"` (remove `CTBValueRestartOnChange` from the check)
- [ ] 1.3 Update `CTBRestartEnabled` to return `true` only for `"true"` (remove `CTBValueRestartOnChange` from the check)
- [ ] 1.4 Update comments on constants to reflect simplified semantics

## 2. Update webhook defaulters

- [ ] 2.1 In `internal/webhook/v1/pod_defaulters.go`, change the `CTBRestartEnabled` check to use the simplified logic (check annotation == `"true"`)
- [ ] 2.2 Update the orphan pod warning message to no longer reference "restart-on-change"

## 3. Update CTB restarter

- [ ] 3.1 In `internal/ctb/restarter.go`, update the comment referencing "restart-on-change" annotation
- [ ] 3.2 Update the orphan pod warning message to no longer reference "restart-on-change"

## 4. Update unit tests

- [ ] 4.1 In `pkg/api/v1/ctb_values_test.go`: remove `"restart-on-change"` test case from `TestCTBMountEnabled`
- [ ] 4.2 In `pkg/api/v1/ctb_values_test.go`: update `TestCTBRestartEnabled` to expect `false` for `"restart-on-change"` and rename/add tests as needed
- [ ] 4.3 In `pkg/api/v1/types_test.go`: update "restart-on-change preserved over all expansion" test
- [ ] 4.4 In `internal/ctb/restarter_test.go`: replace `"restart-on-change"` with `"true"` in all test pods
- [ ] 4.5 In `internal/ctb/watcher_test.go`: replace `"restart-on-change"` with `"true"` in test pod
- [ ] 4.6 In `internal/webhook/v1/pod_webhook_test.go`: replace all `"restart-on-change"` with `"true"`, update test descriptions, remove separate "restart-on-change" mount test (now covered by "true")

## 5. Verify

- [ ] 5.1 Run `go test ./...` to confirm all tests pass
- [ ] 5.2 Run `grep -rn "restart-on-change"` to confirm no references remain in non-test code
- [ ] 5.3 Validate the OpenSpec artifacts with `openspec validate remove-restart-on-change`
