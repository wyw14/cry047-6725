# Bug 复现

## Bug 是什么

关键等级设施处于逾期状态时，提交普通预防性保养记录会绕过状态机并把设施直接标记为正常。

## 如何触发

在 Bug 环境执行：

```bash
go test ./internal/application -run '^TestCriticalOverdueFacilityStaysNonNormalAfterMaintenance$' -count=20
```

## 错误信息

20 次均稳定失败，核心信息为：

```text
critical overdue facility became normal after ordinary maintenance
```
