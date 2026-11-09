# Lab 02 新版执行补充

主题：会话。

具体改动：实现独立cmd/testdriver的认证、reset和connect/send_frame/action/observe操作；绑定实际WebSocket，加入非法origin/超大帧/旧连接测试。

固定验收来源：`fixtures/gates/lab02.json`。

执行：`go run ./cmd/course gate 02`。启动依赖、driver协议和安全边界见 [执行说明](../../docs/EXECUTABLE_GATES.md)。

特别注意：driver不写第二套session逻辑。

保留原tasks中的业务任务；本文件补充必须实现的测试路径，而不是替换成只写文档。对每个固定case提交观察值、对应代码路径和一个能使其失败的改动。人工答辩另计。
