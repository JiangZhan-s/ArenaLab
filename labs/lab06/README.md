# Lab 06：重连与状态同步

## 玩家问题
切网后需要准确回到当前战局，不能漏一步或重复播放一步。

## 前置与时间
通过 Lab 05；阅读已有学习债务。建议8～14小时，答辩与复习另计。

## 范围
遵循 contracts/SEMANTICS.md 的持久化与故障承诺。
不变量：I04 I06 I07。故障案例：G10。

## 先预测
写出正常、重复、非法输入的结果。选择一个故障点，画发生前后的事实。禁止先让Codex填答案。

## 顺序
阅读 [tasks](tasks.md)，固定 [验收](acceptance.json)。实现后用 [review](review.md) 审查，进行 [答辩](interview.md)。参考 [reading](reading.md)。

## 故障实验
客户端停在v3，服务到v8，清理v4，再重连；乱序广播v8/v7。

## Gate
客户端不应用不连续增量；快照后终态hash一致；跨房间拒绝；重试无额外行动。
自动证据不替代teach-back。按 TEACHING.md 完成路径解释和迁移题。

## 交付物
- `internal/recovery/resync.go`
- `cmd/bot-client/`
- `test/integration/resync_test.go`
另交 learning/lab06/{prelab,diff-review,experiment,exit-review}.md。
