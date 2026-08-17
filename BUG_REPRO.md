# Bug 复现

## Bug 是什么

异常复检明确失败后仍被保存为 recovered，关联设施提前离开维修状态，并可继续执行恢复确认。

## 如何触发

在 Bug 环境执行：

```bash
go test ./internal/application -run '^TestFailedReinspectionCannotEnterRecoveryConfirmation$' -count=20
```

## 错误信息

20 次均稳定失败，核心信息为：

```text
failed reinspection entered recovered, want open
```
