# 参考实现：怎样运行，怎样用它学习

## 先理解三个不同的“完成”

1. 课程维护者验证参考实现：本仓库的参考测试、录屏、失败记录。
2. 你完成一个 Lab：自己的预测、代码路径审查、故障实验、复述。
3. 面试项目完成：独立改约束、量化对照、未知故障诊断和间隔复测。

第一项不能自动替代后两项。参考实现是可检查的答案，不是证明你掌握的证书。

## 运行路径

从 WSL 仓库根目录执行。环境见 [SETUP](../SETUP.md)。

| 目的 | 命令 | 通过意味着什么 |
|---|---|---|
| 体验最终目标 | `bash scripts/demo.sh` | Compose 服务启动，不等于玩家旅程完成 |
| 全套机器检查 | `bash scripts/wsl-reference.sh` | 模型参考、规则、29 个固定进程场景和额外回归通过 |
| 单 Lab | `bash scripts/wsl-reference.sh 04` | 相同固定断言，真实服务与数据库 |
| 两个浏览器 | `bash scripts/wsl-browser.sh` | 两个隔离 Chromium context，真实 WS、重连、终局一致 |
| 变异验证 | `bash scripts/wsl-mutations.sh` | 五种故意错误的业务实现被指定断言杀死 |
| 学习支架 | `go test ./...` | 工具支架通过，不包括参考业务 |

参考版构建标签为 `reference`，测试故障控制还需 `testhooks`。发布镜像只有 `reference`，不会编入控制端口。`e2e` 触发真实数据库/进程测试；数据库缺失时失败，不 skip、不回退到内存。

`wsl-reference.sh` 自动启动隔离验证库，在 Go 容器中编译 arena/worker/DS，启动测试驱动并按 Lab 顺序验收。Lab00 读 `reference/model.json`，学习版读 `learning/lab00/model.json`；两者不混用。Lab01 的 14 组规则验收来自固定 fixtures。02–14 的 expected values 只存在判卷器，driver 只接收操作和参数。

## 手工体验两个模式

打开两个页面 http://localhost:8080，分别选账号 P0/P1 后登录。默认口令是仓库公开的本地开发口令，不是生产认证。

回合制：两边点匹配。匹配先后顺序决定槽位，账号 ID 不等于槽位；日志会显示你的槽位。使用方向、攻击、跳过；或勾选自动演示本方回合。点击断开重连，观察版本不回退。40 步或 HP=0 结束；积分异步结算。

独立 DS：上一场已结算后，两边选 Dedicated Server 模式并匹配。两人接入才开始 20Hz 移动，WASD 控制，600 tick 结束。浏览器直连 DS；Go 后台只做分配、票据、实例租约和结果结算。账号按 0/1 配对是本地演示约定，不是假装已实现组队邀请系统。

目前 DS 是三进程、每进程一场的 one-shot 池，不自动回收重置为 ready。连续三场之后 no_capacity 是边界，不是无限容量。扩展任务是引入实例生命周期管理器；不要直接把 finished 实例行改为 ready，否则内存仍保存旧 allocation。真实 UE、Agones、Kubernetes 不在这份 Go 演示的已验证范围。

## 每 Lab 的学习动作

先读 README、tasks、acceptance、execution（00–11）及 contracts，然后：

1. 不看 reference 源码写 prelab，列允许结果与禁止结果。
2. 运行该 Lab 参考 gate，选一条请求，画出连接 → handler → 锁/事务 → commit → ACK/广播路径。
3. 先预测一个改动的影响，再让 Codex 只修改该任务。检查 diff，禁止改固定期望使错误变绿。
4. 在学习分支实现或改约束；参考源码可按章节阅读，不要一次复制全部。
5. 执行失败注入，保存命令、退出码、SHA、实际观察；解释“客户端没收到”与“数据库没提交”的区别。
6. 不开代码回答 review/interview；D+1、D+7、D+21 重测。答不出就回到路径，不继续扩规模。

每次给 Codex 的任务格式：`LabXX / 唯一任务 / 先让我预测 / 可修改文件 / 不准改变的契约和断言 / 实际运行命令 / 交付三道追问`。结束让它更新维护记录；你的预测与反思只能由你填写。

## 源码阅读路线

| Lab | 参考入口 | 最重要的提问 |
|---|---|---|
| 00–01 | rules/reference.go、reference/model.json | 哪个字段是事实，哪个只是展示？ |
| 02–03 | server/server.go、store/session.go、client/client.go | 连接替换、慢读、队列满和 goroutine 退出谁负责？ |
| 04 | store/store.go、migrations | 幂等查询为何先于版本拒绝？commit 失败能否重做？ |
| 05–07 | store/lifecycle.go、ready.go、driver/process 路径 | 取消、准备超时、租约、旧 owner 竞争锁的顺序？ |
| 08 | worker、store.Settle | 积分事实与 Redis 投影为何不能同时当真？ |
| 09–11 | testdriver/load.go、backup.go、settlement.go | 负载机排队是否算延迟？恢复是否真正查了新库？ |
| 12 | realtime、server/realtime.go | 入队 ACK 与 durable ACK 有何不同？落后与执行耗时为何分开？ |
| 13–14 | dedicated、testdriver/dedicated.go、fleet.go、web | 谁可以给出胜者？旧票据、旧实例、被攻破的 DS 分别怎样？ |

表中路径相对于 `internal/reference/`（rules 位于 internal/rules，migrations 位于 db）。

## 证据与故障定位

`artifacts/raw/reference` 保存真实进程日志、每操作观测、开放负载样本；`artifacts/raw/browser` 保存截图、录像、trace、JSON；`artifacts/raw/mutations` 保存故意错误版本的失败日志。原始目录不入 Git，避免视频膨胀和误提交凭据。提交脱敏汇总并保留原始文件的 hash，GitHub Actions 上传独立 artifact。

遇到红灯：先确认工具/容器启动错误还是业务断言失败；再读当前操作日志、服务日志和 SQL 真值。不要先提高超时/放宽预算。性能结果必须同时写硬件、持续时间、到达率、错误率和构建模式；race 构建的数字不能冒充 release 容量。

已知边界与未覆盖场景见 [验证记录](VALIDATION.md) 和 [故障映射](COVERAGE.md)。“可开始学习”不意味着“生产就绪”或“所有结业扩展已完成”。
