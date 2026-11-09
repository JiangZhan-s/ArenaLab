# Lab 08 新版执行补充

主题：结算。

具体改动：并发重放真实worker任务、冲突隔离、Redis清空后重建；独立SQL核账。

固定验收来源：`fixtures/gates/lab08.json`。

执行：`go run ./cmd/course gate 08`。启动依赖、driver协议和安全边界见 [执行说明](../../docs/EXECUTABLE_GATES.md)。

特别注意：结果可靠提交不等于客户端每个tick持久化。

保留原tasks中的业务任务；本文件补充必须实现的测试路径，而不是替换成只写文档。对每个固定case提交观察值、对应代码路径和一个能使其失败的改动。人工答辩另计。
