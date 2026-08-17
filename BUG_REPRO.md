# Bug 复现

## Bug 是什么

同一个保养计划进入新的逾期批次后，调度扫描仍把它识别成旧批次，责任人不会收到新的自动保养待办。

## 如何触发

在 Bug 环境执行：

```bash
go test ./internal/application -run '^TestSchedulerCreatesTodoForEachDueOccurrence$' -count=20
```

## 错误信息

20 次均稳定失败，核心信息为：

```text
each due occurrence needs its own todo, got 1
```
