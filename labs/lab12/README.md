# Lab 12：实时 tick、输入与客户端校正

## 为什么做

控制服务能可靠分配比赛，不代表战斗循环能在50ms预算内推进。用最小二维移动暴露同步问题，不扩展成完整射击游戏。

前置：Lab 11 业务验收与答辩通过。先读 [扩展契约](../../contracts/REALTIME_AND_DS.md)、[岗位边界](../../docs/ROLE_FIT.md)、[执行说明](../../docs/EXECUTABLE_GATES.md)。

学习目标文件可按职责组织。已验证参考路径：`internal/reference/realtime/`、`web/`、`internal/reference/testdriver/realtime.go`。学习分支不要直接用参考通过替代自己的审查和解释。

机器验收：

```text
go run ./cmd/course gate 12
```

参考验收：`bash scripts/wsl-reference.sh 12`。上面的默认 gate 命令需自行启动驱动；参考脚本负责完整编排。通过只代表固定场景，仍需人工 review、原始实验和迁移问答。范围不含 UE 引擎开发、完整 FPS/MOBA、生产规模承诺。

任务见 tasks.md；固定输入和断言见 ../../fixtures/gates/lab12.json。
