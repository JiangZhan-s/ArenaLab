# Lab 05：匹配与唯一分配

## 玩家问题
同一人不能进入两局，等待更短也不能总匹配到悬殊对手。

## 前置与时间
通过 Lab 04；阅读已有学习债务。建议8～14小时，答辩与复习另计。

## 范围
遵循 contracts/SEMANTICS.md 的持久化与故障承诺。
不变量：I01 I02。故障案例：G08 G09。

## 先预测
写出正常、重复、非法输入的结果。选择一个故障点，画发生前后的事实。禁止先让Codex填答案。

## 顺序
阅读 [tasks](tasks.md)，固定 [验收](acceptance.json)。实现后用 [review](review.md) 审查，进行 [答辩](interview.md)。参考 [reading](reading.md)。

## 故障实验
两个matcher同时选择同一人；取消与分配同时提交；连接在返回assignment前断开。

## Gate
无重复占用；取消/分配唯一赢家；已分配结果可查；极端积分玩家等待有报告。
自动证据不替代teach-back。按 TEACHING.md 完成路径解释和迁移题。

## 交付物
- `internal/matchmaking/`
- `db/migrations/000002_matchmaking.sql`
- `test/concurrency/matchmaking_test.go`
- `loadtest/matchmaking/`
另交 learning/lab05/{prelab,diff-review,experiment,exit-review}.md。
