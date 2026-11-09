# Lab 14：双模式可运行交付与 UE 接入里程碑

## 为什么做

把后台能力变成可演示产品，区分独立DS接口已验证与真实UE已联调，给面试官可复查的证据。

前置：Lab 13 业务验收与答辩通过。先读 [扩展契约](../../contracts/REALTIME_AND_DS.md)、[岗位边界](../../docs/ROLE_FIT.md)、[执行说明](../../docs/EXECUTABLE_GATES.md)。

目标文件：`deploy/compose.yaml`、`scripts/demo.ps1`、`scripts/demo.sh`、`docs/evidence/`、`integration/unreal/`。这些是待实现交付物，不表示已经存在。

机器验收：

```text
go run ./cmd/course gate 14
```

初始服务/驱动未实现，应失败。通过只代表固定场景，仍需人工review、原始实验和迁移问答。范围不含UE引擎开发、完整FPS/MOBA、生产规模承诺。

任务见 tasks.md；固定输入和断言见 ../../fixtures/gates/lab14.json。
