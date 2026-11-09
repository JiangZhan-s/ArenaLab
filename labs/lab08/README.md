# Lab 08：终局、积分与排行榜

## 玩家问题
玩家应只得到一次对局积分，缓存损坏不应抹掉历史战绩。

## 前置与时间
通过 Lab 07；阅读已有学习债务。建议8～14小时，答辩与复习另计。

## 范围
遵循 contracts/SEMANTICS.md 的持久化与故障承诺。
不变量：I09 I10 I11。故障案例：G14 G15。

## 先预测
写出正常、重复、非法输入的结果。选择一个故障点，画发生前后的事实。禁止先让Codex填答案。

## 顺序
阅读 [tasks](tasks.md)，固定 [验收](acceptance.json)。实现后用 [review](review.md) 审查，进行 [答辩](interview.md)。参考 [reading](reading.md)。

## 故障实验
积分事务提交后确认前kill；20次投递；清空测试Redis后重建。

## Gate
每人每局一次流水；汇总等于流水；abort不计分；Redis丢失不改变事实。
自动证据不替代teach-back。按 TEACHING.md 完成路径解释和迁移题。

## 交付物
- `internal/settlement/`
- `internal/leaderboard/`
- `cmd/settlement-worker/`
- `db/migrations/000004_settlement.sql`
- `test/process/settlement_test.go`
另交 learning/lab08/{prelab,diff-review,experiment,exit-review}.md。
