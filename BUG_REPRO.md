# Bug 复现

关键设施逾期时提交普通保养会绕过状态机直接回到正常。

```bash
go test ./internal/application -run '^TestCriticalOverdueFacilityStaysNonNormalAfterMaintenance$' -count=1
```
