# ArenaLab：Go 游戏服务课程仓库

这是 Go 游戏后端课程仓库，包含教材、任务、模拟问答、学习支架，以及独立的可运行参考实现、真实进程测试驱动和双浏览器演示。参考实现不等于你已完成学习；不会代填学习者记录。

远程仓库：https://github.com/JiangZhan-s/ArenaLab

目标：先构建可玩的 1v1 回合制竞技服务，再扩展 20 Hz 实时移动实验和独立 Dedicated Server 接入。Go 负责游戏后台与服务端实验，不替代 UE 战斗引擎。Codex 负责主要编码；学习者负责预测、审查、实验、诊断、解释和迁移。

新版先读：[岗位适配边界](docs/ROLE_FIT.md)、[可执行验收](docs/EXECUTABLE_GATES.md)、[量化与面试证据](docs/MEASUREMENT_AND_INTERVIEW.md)、[最终可运行交付](docs/FINAL_DELIVERY.md)。历史项目规划保留作为回合制主线说明，新增实时/DS范围以这些文件及扩展契约为准。

首次学习：

1. [START_HERE](START_HERE.md)：第一节课和环境命令。
2. [项目与课程设计](PROJECT_AND_COURSE.md)：真实场景与架构演进。
3. [规范性业务契约](contracts/SEMANTICS.md)：状态、ACK、幂等与时间边界。
4. [Lab 00](labs/lab00/README.md)：填写自己的初始预测。

## WSL 快速启动

在 WSL 的 Linux 文件系统中 clone（建议 `~/src/ArenaLab`，不要把 Windows 工具链混用到 WSL）。需要 Git、Docker Engine 和 Docker Compose；无需宿主机 Go：

```bash
bash scripts/demo.sh                    # 服务 + MySQL + Redis + 三个 DS
bash scripts/wsl-reference.sh           # 00 模型参考、01 规则、02–14 真实进程验收
bash scripts/wsl-reference.sh 04,07      # 单独重跑事务和 fencing
bash scripts/wsl-browser.sh             # Chromium 两个独立客户端完整演示
```

打开 `http://localhost:18080`，两个页面分别选 P0/P1，登录后匹配。先试回合制，再试独立 DS 模式。所有凭据仅用于本地实验，禁止公网暴露。详见 [运行和学习手册](docs/REFERENCE_GUIDE.md) 与 [验证记录](docs/VALIDATION.md)。

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
| [12](labs/lab12/README.md) | 实时 tick、输入与客户端校正 |
| [13](labs/lab13/README.md) | Dedicated Server 分配与结果鉴权 |
| [14](labs/lab14/README.md) | 双模式交付与 UE 接入里程碑 |

每 Lab 包含 README、tasks、acceptance.json、review、interview、reading 六份材料。00～11 另有 execution.md，补充新版必须完成的执行路径；开始实践必须一起阅读。

## 运行边界

```text
go run ./cmd/course check
go test ./...
go run ./cmd/course gate 01
```

前两条检查支架；第三条是 Lab 01 业务验收，初始应失败，因为 Apply 留给学习者驱动 Codex 实现。
Lab 00 有模型机器小测和独立人工答辩；Lab 02～14 的固定验收连接真实测试驱动比较观察值。参考实现使用 `-tags reference`，普通构建仍走学习入口。不要直接复制参考实现后把绿色验收记成掌握。
绿色 starter CI 不表示业务 Gate 已通过；reference CI、人工答辩、真实 UE 联调分别记录。

量化工具：`go run ./cmd/measure artifacts/raw/run.json`；格式和对照实验见量化文档。不要把示例数据当成绩。真实 UE 联调独立记录；Go ds-sim 通过不等于 UE 接入通过。

## 常用资料

- [学习与评分](TEACHING.md)、[进度](PROGRESS.md)、[协作规则](AGENTS.md)
- [协议指南](contracts/PROTOCOL.md)、[表设计](docs/SCHEMA_GUIDE.md)
- [18 组故障实验](ACCEPTANCE.md)、[易漏细节](docs/IMPORTANT_DETAILS.md)
- [额外面试题](assessments/QUESTION_BANK.md)、[最终答辩](assessments/FINAL_DEFENSE.md)
- [文件职责](docs/REPOSITORY_MAP.md)、[交付边界](docs/COURSE_STATUS.md)
- [环境](SETUP.md)、[提交规范](CONTRIBUTING.md)

PowerShell 可运行 scripts/start-lab.ps1 -Lab 0 创建缺失的学习模板，不覆盖已有回答。
