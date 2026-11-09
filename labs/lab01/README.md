# Lab 01：规则引擎与重放

## 玩家问题
双方必须对一局结果有一致解释，错误操作不能污染后续回合。

## 前置与时间
通过 Lab 00；阅读已有学习债务。建议8～14小时，答辩与复习另计。

## 范围
单进程/内存实验，不承诺崩溃持久性。
不变量：I05 I08 I09。故障案例：G01。

## 先预测
写出正常、重复、非法输入的结果。选择一个故障点，画发生前后的事实。禁止先让Codex填答案。

## 顺序
阅读 [tasks](tasks.md)，固定 [验收](acceptance.json)。实现后用 [review](review.md) 审查，进行 [答辩](interview.md)。参考 [reading](reading.md)。

## 故障实验
当前玩家越界、攻击远处敌人；第40步同时击杀；终局再次动作。

## Gate
固定用例全部通过；非法动作无副作用；终局和40回合边界有测试。
自动证据不替代teach-back。按 TEACHING.md 完成路径解释和迁移题。

## 交付物
- `internal/rules/rules.go`
- `internal/rules/replay_test.go`
- `cmd/replay/main.go`
另交 learning/lab01/{prelab,diff-review,experiment,exit-review}.md。
