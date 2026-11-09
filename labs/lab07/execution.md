# Lab 07 新版执行补充

主题：所有权。

具体改动：真实暂停旧owner并启新owner，再恢复旧进程；另测abort/no-score。两种策略分别报告保证和成本。

固定验收来源：`fixtures/gates/lab07.json`。

执行：`go run ./cmd/course gate 07`。启动依赖、driver协议和安全边界见 [执行说明](../../docs/EXECUTABLE_GATES.md)。

特别注意：本Lab回合制恢复不得外推为实时战斗无损迁移。

保留原tasks中的业务任务；本文件补充必须实现的测试路径，而不是替换成只写文档。对每个固定case提交观察值、对应代码路径和一个能使其失败的改动。人工答辩另计。
