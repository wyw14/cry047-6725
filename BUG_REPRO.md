# Bug 复现

失败复检被错误地标记为 recovered，因而可以继续执行恢复确认。

```bash
go test ./internal/application -run '^TestFailedReinspectionCannotEnterRecoveryConfirmation$' -count=1
```
