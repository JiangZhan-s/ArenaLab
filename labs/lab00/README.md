# Lab 00：模型与业务承诺

## 玩家问题
玩家的攻击到底何时算成功？只有先定义承诺，才知道哪些故障是bug。

## 前置与时间
从 START_HERE 开始。建议4～6小时，答辩与复习另计。

## 范围
单进程/内存实验，不承诺崩溃持久性。
不变量：I04 I07 I08 I09。故障案例：设计序列。

## 先预测
写出正常、重复、非法输入的结果。选择一个故障点，画发生前后的事实。禁止先让Codex填答案。

## 顺序
阅读 [tasks](tasks.md)，固定 [验收](acceptance.json)。实现后用 [review](review.md) 审查，进行 [答辩](interview.md)。参考 [reading](reading.md)。

## 故障实验
客户端上报胜利→断网→重新登录→重放行动，逐步标记已知与未知事实。

## Gate
四张转移表一致；12条不变量均有一个反例和未来测试；十个序列有允许终态。
自动证据不替代teach-back。按 TEACHING.md 完成路径解释和迁移题。

## 交付物
- `docs/state-machines.md`
- `docs/invariants.md`
- `docs/failure-model.md`
另交 learning/lab00/{prelab,diff-review,experiment,exit-review}.md。
