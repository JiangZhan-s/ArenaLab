# Lab 04 新版执行补充

主题：持久化。

具体改动：将外部kill屏障放在commit前及commit后ACK前；重启真实进程，以独立SQL连接核对事件/命令/版本。

固定验收来源：`fixtures/gates/lab04.json`。

执行：`go run ./cmd/course gate 04`。启动依赖、driver协议和安全边界见 [执行说明](../../docs/EXECUTABLE_GATES.md)。

特别注意：不能通过return error冒充进程崩溃。

保留原tasks中的业务任务；本文件补充必须实现的测试路径，而不是替换成只写文档。对每个固定case提交观察值、对应代码路径和一个能使其失败的改动。人工答辩另计。
