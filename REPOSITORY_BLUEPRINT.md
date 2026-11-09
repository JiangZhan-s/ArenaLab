# 课程仓库蓝图与建设顺序

## 建议新仓库

仓库名称 ArenaLab，远端为 https://github.com/JiangZhan-s/ArenaLab 。本文保留目标布局；当前课程骨架已经建立，实际文件与待实现边界见 docs/REPOSITORY_MAP.md 和 docs/COURSE_STATUS.md。清单采用 JSON 以便 Go 标准库校验。

```text
arenalab/
├── README.md
├── START_HERE.md
├── AGENTS.md
├── PROGRESS.md
├── course.yaml
├── Makefile
├── go.mod
├── .github/workflows/ci.yml
├── contracts/
│   ├── states.yaml
│   ├── invariants.yaml
│   ├── http.openapi.yaml
│   ├── websocket.schema.json
│   └── events.schema.json
├── cmd/
│   ├── arena-server/
│   ├── settlement-worker/
│   ├── bot-client/
│   └── replay/
├── internal/
│   ├── rules/
│   ├── protocol/
│   ├── auth/
│   ├── session/
│   ├── matchmaking/
│   ├── room/
│   ├── ownership/
│   ├── recovery/
│   ├── settlement/
│   ├── leaderboard/
│   ├── observability/
│   └── testkit/
├── db/migrations/
├── deploy/
├── web/                    最小演示客户端，后期增加
├── labs/lab00...lab11/
├── test/
│   ├── acceptance/
│   ├── integration/
│   ├── concurrency/
│   ├── process/
│   ├── security/
│   ├── replay/
│   └── mutation/
├── fixtures/
├── faults/
├── loadtest/
├── learning/labXX/
├── assessments/
├── templates/
└── docs/
    ├── decisions/
    ├── experiments/
    ├── runbooks/
    └── postmortems/
```

## 每个 Lab 的文件合同

- README.md：为什么需要、业务边界、前置、实验步骤和 Gate。
- tasks.md：分为可审查的小任务，列出改动目录和禁止提前引入的组件。
- acceptance.yaml：关联不变量 ID、故障 ID、测试命令、预期事实。
- review.md：关键路径、并发交错、事务边界、资源退出检查点。
- interview.md：公开问题和追问；答案要点折叠或先由 Examiner 保留，首次作答前不展示。
- reading.md：官方资料的具体章节及阅读问题，不依赖旧会话。

learning/labXX/ 下由学习者保存 prelab、diff-review、experiment、exit-review 和复习记录；Codex 不代填个人预测与反思。

## 首批数据模型

Lab 04：matches、match_players、match_states、match_events、command_results。事件键 (match_id, state_version)，命令键 (match_id, player_id, command_id)，同时保留 payload hash。state 更新和 event、command result 同事务。

Lab 05：matchmaking_tickets、player_occupancy、assignments。player_occupancy 主键 player_id，避免仅靠状态过滤查询保护唯一占用。匹配事务按 player_id 顺序锁定，取消/分配竞争要显式串行化。

Lab 07：room_owners，保存 owner、epoch、lease_until。状态写事务与 owner 行协调，不能先在事务外查租约再写战局。

Lab 08：match_results、settlement_jobs、rating_ledger、player_ratings。规则版本和赛季进入终局事实；积分流水与汇总同事务。投影使用结果 ID 去重或版本化替换，不能重复 INCR 后再记录已消费。

快照引入 snapshot_version、state_hash、rules_version 和日志保留水位；清理前验证恢复可用。主线不建设虚拟货币、抽卡、商城、好友、公会和聊天。

## 命令合同

未来提供 verify、test-rules、test-integration、test-process、test-race、test-mutation、replay、load 四类独立入口和对应 PowerShell 入口。

CI 初期检查规则、协议 fixtures、格式和静态分析；进入 MySQL 阶段增加真实数据库验收；进程与压测套件可定时或按需运行，但必须记录结果。尚未实现的 gate 不得用 skip 伪装全通过。

## 建设里程碑

M0：统一状态/协议/不变量，生成完整课程 handout 和验收 fixtures；暂不提供核心业务答案。

M1：单进程可玩的双人对局，完成 Lab 00～04，保证行动 ACK 与持久化语义。

M2：匹配、重连、多实例恢复，完成 Lab 05～07。

M3：结算、排行和性能实验，完成 Lab 08～09。

M4：发布恢复演练和最终答辩，完成 Lab 10～11。

各里程碑先修订验收契约再生成实现；课程基础设施可预先创建，学习者业务实现按 Gate 逐步推进。课程设计迭代和学习者作答严格分开。

## 首次学习会话

先让学习者回答：断网后重试一次攻击为何可能攻击两次？单房间单 goroutine 为什么挡不住旧 owner？已收到 ACK 的行动重启后丢失是否可以接受？两个匹配 worker 怎样避免把同一人分进两局？

保留原始回答后，让 Codex 提问、给反例并整理状态表；只有 Gate 00 通过才开始规则实现。仍遵循 Predict → Diff Review → Inject → Retell。
