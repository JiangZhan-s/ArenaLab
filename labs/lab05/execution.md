# Lab 05 新版执行补充

主题：匹配。

具体改动：同一玩家并发入队、重复request_id、取消后分配；用SQL占用表核验。提前保留build/region扩展边界。

固定验收来源：`fixtures/gates/lab05.json`。

执行：`go run ./cmd/course gate 05`。启动依赖、driver协议和安全边界见 [执行说明](../../docs/EXECUTABLE_GATES.md)。

特别注意：cancel与assignment竞争还需随机交错测试。

保留原tasks中的业务任务；本文件补充必须实现的测试路径，而不是替换成只写文档。对每个固定case提交观察值、对应代码路径和一个能使其失败的改动。人工答辩另计。
