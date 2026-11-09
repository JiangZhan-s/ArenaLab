# Lab 09：负载与排障

## 玩家问题
在线连接数不是活跃对局数；高峰重连可能比正常行动更危险。

## 前置与时间
通过 Lab 08；阅读已有学习债务。建议8～14小时，答辩与复习另计。

## 范围
遵循 contracts/SEMANTICS.md 的持久化与故障承诺。
不变量：I12。故障案例：G07 G18。

## 先预测
写出正常、重复、非法输入的结果。选择一个故障点，画发生前后的事实。禁止先让Codex填答案。

## 顺序
阅读 [tasks](tasks.md)，固定 [验收](acceptance.json)。实现后用 [review](review.md) 审查，进行 [答辩](interview.md)。参考 [reading](reading.md)。

## 故障实验
连接数相同，分别空闲/活跃；数据库延迟增长；重连同时爆发。

## Gate
报告offered/completed/rejected、p99、队列/内存/DB等待；过载后可恢复；不变量仍成立。
自动证据不替代teach-back。按 TEACHING.md 完成路径解释和迁移题。

## 交付物
- `loadtest/`
- `internal/observability/`
- `docs/experiments/capacity.md`
- `test/load/overload_test.go`
另交 learning/lab09/{prelab,diff-review,experiment,exit-review}.md。
