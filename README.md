# ArenaLab：Go 游戏服务课程仓库

这是可以开始逐 Lab 学习的课程仓库，包含教材、任务、模拟问答、Starter Code、固定规则验收和学习记录模板。业务实现按 Lab 推进；完整游戏服务尚未实现。

远程仓库：https://github.com/JiangZhan-s/ArenaLab

目标：构建服务器权威的 1v1 回合制竞技服务，通过匹配、长连接、房间调度、重连、战局恢复和结算学习 Go 后端。Codex 负责主要编码；学习者负责预测、审查、实验、诊断、解释和迁移。

首次学习：

1. [START_HERE](START_HERE.md)：第一节课和环境命令。
2. [项目与课程设计](PROJECT_AND_COURSE.md)：真实场景与架构演进。
3. [规范性业务契约](contracts/SEMANTICS.md)：状态、ACK、幂等与时间边界。
4. [Lab 00](labs/lab00/README.md)：填写自己的初始预测。

## 课程导航

| Lab | 主题 |
|---|---|
| [00](labs/lab00/README.md) | 模型与业务承诺 |
| [01](labs/lab01/README.md) | 规则引擎与重放 |
| [02](labs/lab02/README.md) | 会话与 WebSocket |
| [03](labs/lab03/README.md) | 房间并发与资源隔离 |
| [04](labs/lab04/README.md) | 事务、ACK 与幂等 |
| [05](labs/lab05/README.md) | 匹配与分配 |
| [06](labs/lab06/README.md) | 重连与同步 |
| [07](labs/lab07/README.md) | 所有权与恢复 |
| [08](labs/lab08/README.md) | 结算与排行榜 |
| [09](labs/lab09/README.md) | 容量与排障 |
| [10](labs/lab10/README.md) | 发布与数据恢复 |
| [11](labs/lab11/README.md) | 综合事故与答辩 |

每 Lab 包含 README、tasks、acceptance.json、review、interview、reading 六份材料。

## 运行边界

```text
go run ./cmd/course check
go test ./...
go run ./cmd/course gate 01
```

前两条检查支架；第三条是 Lab 01 业务验收，初始应失败，因为 Apply 留给学习者驱动 Codex 实现。
Lab 02～11 现有明确测试契约与交付物，业务自动测试须随实践创建；gate 命令显式报告未实现。
绿色 starter CI 不表示业务 Gate 已通过。当前不包含可玩的游戏服务或压测成绩。

## 常用资料

- [学习与评分](TEACHING.md)、[进度](PROGRESS.md)、[协作规则](AGENTS.md)
- [协议指南](contracts/PROTOCOL.md)、[表设计](docs/SCHEMA_GUIDE.md)
- [18 组故障实验](ACCEPTANCE.md)、[易漏细节](docs/IMPORTANT_DETAILS.md)
- [额外面试题](assessments/QUESTION_BANK.md)、[最终答辩](assessments/FINAL_DEFENSE.md)
- [文件职责](docs/REPOSITORY_MAP.md)、[交付边界](docs/COURSE_STATUS.md)
- [环境](SETUP.md)、[提交规范](CONTRIBUTING.md)

PowerShell 可运行 scripts/start-lab.ps1 -Lab 0 创建缺失的学习模板，不覆盖已有回答。
