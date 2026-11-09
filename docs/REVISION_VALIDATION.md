# 本轮课程改版验证记录（2026-11-09）

这是支架/验收器验证，不是游戏业务验收。
这是参考实现加入之前的历史记录；当前业务验证见 [VALIDATION](VALIDATION.md)，不要把本文的当时红灯状态当成当前参考版状态。

- Windows Go 1.26.5；模块最低版本仍为1.25。
- `go test ./... -timeout=60s`：通过，包括验收器错误观测、并发、数值区间、错误HTTP/JSON和样本汇总测试。
- `go vet ./...`：通过。
- `go run ./cmd/course check`：15 Lab及13份固定驱动场景格式检查通过。
- 125份Markdown相对链接检查、34份JSON解析、`git diff --check`：通过。
- `go run ./cmd/measure fixtures/measurement-example.json`：合成样本计算通过；不属于真实性能成绩。
- `go run ./cmd/course gate-all`：15/15红灯，符合当前未实现状态。00缺学习者模型答案；01规则stub；02～14刻意连接未运行的loopback驱动，均失败，未误标通过。
- 实际游戏服务、真实driver、浏览器旅程、Linux race、UE联调与真实性能实验：本轮未验证，不可标passed。

13份驱动套件共29个固定场景；它们约束驱动/服务应实现的行为，不代表完整业务覆盖。后续必须补原G01～G18故障与随机交错，并审查driver真实性。
