# Lab 11：综合事故与面试交付

## 玩家问题
要证明你能把玩家旅程串起来，并在陌生故障中找到证据。

## 前置与时间
通过 Lab 10；阅读已有学习债务。建议8～14小时，答辩与复习另计。

## 范围
遵循 contracts/SEMANTICS.md 的持久化与故障承诺。
不变量：I01 I02 I03 I04 I05 I06 I07 I08 I09 I10 I11 I12。故障案例：G01 G02 G03 G04 G05 G06 G07 G08 G09 G10 G11 G12 G13 G14 G15 G16 G17 G18。

## 先预测
写出正常、重复、非法输入的结果。选择一个故障点，画发生前后的事实。禁止先让Codex填答案。

## 顺序
阅读 [tasks](tasks.md)，固定 [验收](acceptance.json)。实现后用 [review](review.md) 审查，进行 [答辩](interview.md)。参考 [reading](reading.md)。

## 故障实验
匹配完成→行动提交后断连→重连→owner接管→终局重复结算。

## Gate
玩家终态一致、无重复积分、证据可重现；能闭卷定位事务与故障窗口。
自动证据不替代teach-back。按 TEACHING.md 完成路径解释和迁移题。

## 交付物
- `test/process/capstone_test.go`
- `docs/experiments/final.md`
- `docs/postmortems/capstone.md`
- `learning/lab11/exit-review.md`
另交 learning/lab11/{prelab,diff-review,experiment,exit-review}.md。
