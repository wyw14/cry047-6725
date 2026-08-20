# Bug 复现

同一计划跨到新的逾期批次后，自动待办被错误地按旧批次去重。

```bash
go test ./internal/application -run '^TestSchedulerCreatesTodoForEachDueOccurrence$' -count=1
```
