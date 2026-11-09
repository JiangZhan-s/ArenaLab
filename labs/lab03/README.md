# Lab 03：房间循环与资源隔离

## 玩家问题
慢读玩家不能拖住其他对局或吃光内存。

## 前置与时间
通过 Lab 02；阅读已有学习债务。建议8～14小时，答辩与复习另计。

## 范围
单进程/内存实验，不承诺崩溃持久性。
不变量：I05 I12。故障案例：G03 G07。

## 先预测
写出正常、重复、非法输入的结果。选择一个故障点，画发生前后的事实。禁止先让Codex填答案。

## 顺序
阅读 [tasks](tasks.md)，固定 [验收](acceptance.json)。实现后用 [review](review.md) 审查，进行 [答辩](interview.md)。参考 [reading](reading.md)。

## 故障实验
同时投递同version两行动；一玩家停止读socket并持续产生广播。

## Gate
最多一个同版本行动有效；所有队列有上限；另一房间可继续；退出测试有期限。
自动证据不替代teach-back。按 TEACHING.md 完成路径解释和迁移题。

## 交付物
- `internal/room/`
- `test/concurrency/room_test.go`
- `test/integration/slow_reader_test.go`
另交 learning/lab03/{prelab,diff-review,experiment,exit-review}.md。
