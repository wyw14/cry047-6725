# Bug 复现

调整周期后，初始计划版本的历史周期被覆盖成新周期。

```bash
go test ./internal/application -run '^TestChangingCyclePreservesHistoricalVersionSnapshot$' -count=1
```
