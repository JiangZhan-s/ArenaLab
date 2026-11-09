# 原故障矩阵与参考测试的实际覆盖

固定 Gate 的测试是可执行的下限，不是覆盖全部需求的证明。`passed` 的具体运行见 VALIDATION.md；此表描述测试范围。

| 原项 | 当前路径 | 尚未覆盖的部分 |
|---|---|---|
| G01 非当前玩家 | Lab01 固定规则验收 | 随机性质序列可扩展 |
| G02 幂等重放/冲突 | Lab04 dedup-before-version、新会话原 ID 重试 | 20 次与不同并发排列应追加 |
| G03 同版本竞争 | Lab03 same-version-concurrency，独立 SQL | 任意交错不可能穷举 |
| G04/G05 commit 前后 kill | Lab04 真实屏障 + SIGKILL + 重启 | DB 自身崩溃、磁盘故障不覆盖 |
| G06 旧会话 | Lab02 替换后写拒绝，额外一次性 WS 票据回归 | 跨地域认证不覆盖 |
| G07 慢读 | Lab03 真实 socket 慢读、队列峰值、健康房间延迟、清理 | 极限 RSS 与跨网络实验另做 |
| G08/G09 匹配/取消 | Lab05 重复排队单占用、已取消不分配 | 双 matcher 和 cancel/assign 同步 barrier 竞争未覆盖 |
| G10 重连 | Lab06 全量快照回退，新 epoch 重试原命令 | delta 路径未实现，全量快照是当前基线 |
| G11 owner fencing | Lab07 quiesce 后真实 STOP/CONT、数据库租约接管 | 任意 SQL 临界区内 STOP 不保证 5s 恢复，见 ADR0002 |
| G12/G13 时间边界 | 实现锁后读取 DB 时间、一次当前回合 timeout；准备超时回归 | 等于 deadline 的可控 DB 时钟、长停顿矩阵未覆盖 |
| G14 结算 | Lab08 多 worker 重放、冲突拒绝、SQL 流水唯一 | worker 每个事务位置 kill 与持久冲突告警队列未覆盖 |
| G15 投影丢失 | Lab08 清理随机命名空间、实际 Redis 重建、旧版本拒绝 | Redis HA 故障切换不覆盖 |
| G16 drain | Lab10 新请求拒绝、既有对局可继续；Lab14 DS drain/故障隔离 | ready 池有额外空闲容量时的定向 drain 竞争需补充 |
| G17 规则升级 | 每局固定 rules_version、保存事件快照 | 第二规则版本与混部兼容实验未实现 |
| G18 过载 | Lab09 开放负载、真实行锁造成过载、恢复；Lab12 SQL 延迟 | 大规模重连风暴、连接数容量、全链路网络故障另做 |

实时扩展：Lab12 验证输入序号、限速、预测校正、200 个真实 tick 和 SQL 隔离；调度器单测验证最多补 3 tick 与丢弃落后计数。注入 RTT/jitter 是驱动的确定性消息调度，不是声称实测公网或 Linux tc 网络。

DS 扩展：Lab13/14 验证真实独立进程、分配幂等、票据过期/一次消费、服务身份、旧 generation、双客户端终局与失败实例 no-score。额外回归在清理器尚未运行时拒绝过期心跳结果。真实 UE 为 not_run。

这些缺口是进阶学习任务，不要改原契约来把未覆盖写成已覆盖，也不要在简历中把29个固定场景写成全部生产故障已验证。
