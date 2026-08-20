# Bug 复现

执行证据的切片和数值指针未隔离，请求方后续修改会污染已提交留证。

```bash
go test ./internal/application -run '^TestSubmittedExecutionEvidenceIsDetached$' -count=1
```
