# Lab 04：事务、ACK与幂等

## 玩家问题
玩家看到行动成功后，应用进程崩溃不应让行动消失。

## 前置与时间
通过 Lab 03；阅读已有学习债务。建议8～14小时，答辩与复习另计。

## 范围
遵循 contracts/SEMANTICS.md 的持久化与故障承诺。
不变量：I04 I06 I07。故障案例：G02 G04 G05 G12。

## 先预测
写出正常、重复、非法输入的结果。选择一个故障点，画发生前后的事实。禁止先让Codex填答案。

## 顺序
阅读 [tasks](tasks.md)，固定 [验收](acceptance.json)。实现后用 [review](review.md) 审查，进行 [答辩](interview.md)。参考 [reading](reading.md)。

## 故障实验
同command20次，异payload复用ID；kill后重启并查询持久化事实。

## Gate
三个事实原子一致；commit后重试不再行动；commit前不返回committed；deadline相等拒绝。
自动证据不替代teach-back。按 TEACHING.md 完成路径解释和迁移题。

## 交付物
- `db/migrations/000001_match.sql`
- `internal/persistence/`
- `test/integration/command_test.go`
- `test/process/action_crash_test.go`
另交 learning/lab04/{prelab,diff-review,experiment,exit-review}.md。
