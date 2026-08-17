# Bug 复现

## Bug 是什么

调整保养周期时，旧的计划版本快照被新周期覆盖，审计历史无法还原调整前的真实配置。

## 如何触发

在 Bug 环境执行：

```bash
go test ./internal/application -run '^TestChangingCyclePreservesHistoricalVersionSnapshot$' -count=20
```

## 错误信息

20 次均稳定失败，核心信息为：

```text
initial version was rewritten from 30 to 60
```
