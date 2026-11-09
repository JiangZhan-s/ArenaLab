# Lab 10：发布、版本与数据恢复

## 玩家问题
更新规则和进程时，已开局玩家不能无解释地丢局。

## 前置与时间
通过 Lab 09；阅读已有学习债务。建议8～14小时，答辩与复习另计。

## 范围
遵循 contracts/SEMANTICS.md 的持久化与故障承诺。
不变量：I06 I07 I09。故障案例：G16 G17。

## 先预测
写出正常、重复、非法输入的结果。选择一个故障点，画发生前后的事实。禁止先让Codex填答案。

## 顺序
阅读 [tasks](tasks.md)，固定 [验收](acceptance.json)。实现后用 [review](review.md) 审查，进行 [答辩](interview.md)。参考 [reading](reading.md)。

## 故障实验
drain与分配竞争；新版代码读取旧局；快照损坏；备份恢复后对账。

## Gate
无新分配落到draining；旧局可解释；恢复库通过事实核对；报告RPO/RTO测量。
自动证据不替代teach-back。按 TEACHING.md 完成路径解释和迁移题。

## 交付物
- `internal/lifecycle/`
- `docs/runbooks/release.md`
- `docs/runbooks/restore.md`
- `test/process/drain_test.go`
另交 learning/lab10/{prelab,diff-review,experiment,exit-review}.md。
