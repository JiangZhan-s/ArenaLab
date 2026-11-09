# 参考实现验证记录（2026-11-09）

这份记录属于课程维护，不代表学习者已经完成 Lab。最终可复现命令见 REFERENCE_GUIDE.md；具体未覆盖项见 COVERAGE.md。原始文件位于 artifacts/raw，默认不入 Git。

## 已执行的参考验证

| 项目 | 结果 | 实际范围 |
|---|---|---|
| Lab00 | passed | reference/model.json 的机器小测；人工状态表未代填 |
| Lab01 | passed | 14 个固定规则 fixture，reference + acceptance 构建 |
| Lab02–14 | passed | 29 个固定场景，真实 MySQL/Redis/WS/子进程，完整连续运行 |
| race / vet | passed | 服务、worker、DS 与 e2e driver 使用 race；reference vet |
| 额外回归 | passed | 准备超时释放占用、未清理时拒绝过期 DS 结果、WS 一次性票据 |
| 调度器 | passed | 最多补 3 tick，丢弃落后计数与重新调度 |
| 五个变异 | killed 5/5 | 不提交就成功、版本先于幂等、去掉 fencing、结算不幂等、过期票据放行 |
| 新库 Compose 启动 | passed | 新的 arenalab-fresh 卷初始化，7 服务启动，release 无 testhooks |
| 原生 Chromium 双页面 | passed | 回合制 version41、DS tick600，重连后终局一致，无页面脚本错误 |
| 真实 Unreal | not_run | Go DS 通过不等于 UE Replication、引擎 DS 或 UE 客户端联调 |
| 学习者答辩/三组 A/B/第二人复现 | not_run | 留给学习者的个人结业证据，不由维护工作代填 |

完整成功运行之一：TestRealProcessGates 183.87s；附加回归 0.68s，总 e2e 测试包 185.579s。该数字是测试耗时，不是游戏服务吞吐或恢复 SLA。浏览器成功记录时间 09:21:42–09:22:19 UTC。以上为提交前工作树验证，不能冒充某个已发布 SHA 的结果；最终精确 SHA 与原始文件 hash 另附证据摘要。

## 失败与修复（保留反例）

- 首次 Lab07 真实 STOP 可能冻结持锁 SQL 事务，租约过期不释放行锁。按 ADR0002 明确暂停位置；任意事务中 STOP 的恢复时限仍未证明。
- 浏览器首次失败：将账号 ID 当成匹配槽位，两个页面都收到 not_turn。改为 join 返回 player_slot；UI 使用服务器分配槽位。说明纯协议 bot 通过不能替代浏览器旅程。
- 一次全量 race 捕获 client 缓存与返回响应共用 map。修复为深拷贝，后续全量 race 通过。
- 新增准备超时清理器最初产生空事务，使 Lab12 SQL commit 断言失败；改为无候选不启动事务，锁内仍重新检查过期条件。没有放宽断言。
- `/mnt/c` npm 文件损坏：改用 ext4 临时依赖和 Docker node_modules volume，未手改第三方包。
- WSL 无前台进程自动停止导致后续连接拒绝：增加 `--attach` 使用方式；持久演示卷保留数据。验证库 tmpfs 则按设计丢弃。

## 不能据此宣称

不能写“百万在线”“商业 FPS/MOBA 服务端”“完成 Unreal 引擎开发”“生产级”“全部 G01–G18 故障已验证”。可以写“Go 游戏后端课程参考实现，事务/幂等/fencing/结果鉴权的可复现实验，20Hz 独立进程同步演示”。进一步的个人面试成果需要你自己解释、改变约束和做重复对照。

参考版当前限制：公开本地开发凭据，单数据库，全量快照，三实例 one-shot 池，固定配对，DS 失效 no-score，未实现规则双版本混部、生产冲突告警闭环、实例自动回收、数据库 HA/PITR 或公网安全边界。保留这些限制比虚构生产经历更有面试价值。
