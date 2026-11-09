# ArenaLab 验收与故障矩阵

这是结业故障矩阵。已有可运行参考服务与固定验收；逐项实际覆盖见 docs/COVERAGE.md。固定 Gate 通过不代表本矩阵所有交错均已覆盖，不能因规划文件存在而标为通过。

## 状态模型

- Ticket：queued → assigned / cancelled / expired。终态不再重新入队；重新排队创建新票据。
- Match：allocated → waiting_ready → running → finished / aborted。准备超时或恢复不可行可 aborted；finished 不可撤销。
- Connection：authenticated → attached → replaced / closed。连接退出不等于玩家离开战局。
- Settlement：pending → applied；永久异常进入 quarantined 并告警，不静默吞掉。

每个状态表还需在 Lab 00 补全事务边界、触发身份和拒绝条件。ArenaLab 的远端仓库保存本规划；OrderGuard 独立保留。

## 固定实验

| ID | 注入点/输入 | 必须观察到的结果 | 未来测试 |
|---|---|---|---|
| G01 | 非当前玩家 action | 拒绝，版本/回合/生命不变 | rules_test |
| G02 | 同 command_id 重放 20 次 | 一次行动，重放同结果；异 payload 冲突 | command_dedup_test |
| G03 | 同 expected version 两条合法命令竞争 | 最多一个成功 | concurrent_action_test |
| G04 | action 事务 commit 前 kill | 没有 committed ACK；恢复后可重试 | action_crash_test |
| G05 | action commit 后 ACK 前 kill | 状态可恢复，重试不再行动 | action_crash_test |
| G06 | 旧连接在新 epoch 登录后继续发送 | 旧连接无权提交；新连接可同步 | session_replacement_test |
| G07 | 玩家停止读 socket | 有界队列、超限断连；另一房间仍前进 | slow_reader_test |
| G08 | 双 matcher 对同一票据组分配 | 玩家只在一局内；输家重读状态 | matchmaking_race_test |
| G09 | cancel 与 assignment 同时提交 | 只有一种允许终态；客户端查询可确认结果 | ticket_cancel_race_test |
| G10 | 重连版本落后且日志已清理 | 返回快照，之后增量连续，终态 hash 一致 | resync_test |
| G11 | owner A 暂停、B 接管、A 恢复 | A 写入被 fencing 拒绝，B 唯一推进 | owner_takeover_test |
| G12 | deadline 相等、之前、之后提交 | 严格遵守事务内边界，无双回合事件 | deadline_test |
| G13 | server 停顿超过多个回合长度 | 只处理当前回合超时，按恢复规则建立新 deadline | recovery_clock_test |
| G14 | 终局提交后 worker 崩溃/重复结算 | 结果不丢；每玩家一次积分流水 | settlement_crash_test |
| G15 | Redis 全部丢失 | 计分事实不变，可从持久化流水重建排行 | leaderboard_rebuild_test |
| G16 | drain 期间新匹配分配 | 不向 draining 实例新分配；现有局按策略结束或恢复 | drain_test |
| G17 | 规则版本升级后重放旧局 | 使用旧版本规则或保存的事件解释，不能误用新规则 | replay_compat_test |
| G18 | socket 洪泛、DB 变慢、重连潮 | 队列/连接/内存有界，拒绝明确，负载下降后恢复 | overload_test |

## 独立判定器

测试观测必须包括持久化事件、命令结果、state_version、owner_epoch、积分流水，而非只检查 HTTP/WS 成功码。

纯规则提供至少 10 组固定输入与预期状态 fixtures；随机性质测试记录 seed。并发测试用 barrier，期限测试用可控时钟；进程故障使用真实子进程，不能用返回 error 代替断电/kill 窗口。

为以下错误实现做变异测试：先 ACK 后写库、不校验 epoch、幂等先查后写、慢连接无界队列、终局每次重试都计分。判定器必须使其失败。学习者可以看验收规格；迁移题先答后看要点，不能把同一套题背熟当作掌握。

## 性能指标

- 同时连接数不等于活跃玩家数；活跃房间、每房间每秒命令量、消息大小必须一起报告。
- 指标：行动到 committed ACK 的 p50/p95/p99、匹配等待 p95、每局恢复时间、queue wait、队列深度、DB pool wait、goroutine、CPU、内存、网络出站流量。
- 匹配同时报告积分差、等待长尾和饥饿；排行报告投影延迟。
- bot 支持 idle、active、slow_reader、reconnect_storm 四种模型。
- 固定硬件、版本、数据库持久化参数，保存 commit、负载配置和原始结果。
- 先测容量拐点，再围绕该点设置目标；不得事先编造“十万在线”。负载发生器也要测自身 CPU 与连接上限。

## 结业边界

可演示系统：玩家旅程闭环，有 bot 和可复现命令。

优秀面试项目：G01～G18 有证据，至少三个设计专题能闭卷解释，有一次未知故障排查与间隔复测。

生产就绪评估：在具体部署上另外检查身份/权限、密钥、网络边界、数据库恢复、容量目标、监控告警、发布与回滚；个人课程验收不能替代上线评审。
