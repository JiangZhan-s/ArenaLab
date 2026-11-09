# Lab 13：Dedicated Server 分配与结果信任边界

## 为什么做

FPS/MOBA或UE比赛服与通用Go后台分离。后台必须避免重复分配、旧实例上报和玩家伪造结果，不需要自己写UE网络栈。

前置：Lab 12 业务验收与答辩通过。先读 [扩展契约](../../contracts/REALTIME_AND_DS.md)、[岗位边界](../../docs/ROLE_FIT.md)、[执行说明](../../docs/EXECUTABLE_GATES.md)。

目标文件：`cmd/ds-sim/`、`internal/allocation/`、`internal/tickets/`、`testdriver/dedicated.go`。这些是待实现交付物，不表示已经存在。

机器验收：

```text
go run ./cmd/course gate 13
```

初始服务/驱动未实现，应失败。通过只代表固定场景，仍需人工review、原始实验和迁移问答。范围不含UE引擎开发、完整FPS/MOBA、生产规模承诺。

任务见 tasks.md；固定输入和断言见 ../../fixtures/gates/lab13.json。
