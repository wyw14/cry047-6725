# Bug 复现

## Bug 是什么

执行记录保存后仍与请求及响应对象共享检查值、照片和耗材的可变内存，调用方后续修改会污染已经提交的留证。

## 如何触发

在 Bug 环境执行：

```bash
go test ./internal/application -run '^TestSubmittedExecutionEvidenceIsDetached$' -count=20
```

## 错误信息

20 次均稳定失败，核心信息为：

```text
inspection evidence changed after submit
```
