# Lab 07：所有权与战局恢复

## 玩家问题
旧服务器暂停后恢复，可能和接管节点同时处理同一局。

## 前置与时间
通过 Lab 06；阅读已有学习债务。建议8～14小时，答辩与复习另计。

## 范围
遵循 contracts/SEMANTICS.md 的持久化与故障承诺。
不变量：I03 I06 I07。故障案例：G11 G12 G13。

## 先预测
写出正常、重复、非法输入的结果。选择一个故障点，画发生前后的事实。禁止先让Codex填答案。

## 顺序
阅读 [tasks](tasks.md)，固定 [验收](acceptance.json)。实现后用 [review](review.md) 审查，进行 [答辩](interview.md)。参考 [reading](reading.md)。

## 故障实验
A持旧epoch，B已提交下一步；A恢复尝试写及发延迟广播。

## Gate
旧owner不能提交；数据库不可达不ACK；新owner加载一致；超时仅推进当前一回合。
自动证据不替代teach-back。按 TEACHING.md 完成路径解释和迁移题。

## 交付物
- `internal/ownership/`
- `internal/recovery/`
- `db/migrations/000003_ownership.sql`
- `test/process/owner_takeover_test.go`
另交 learning/lab07/{prelab,diff-review,experiment,exit-review}.md。
