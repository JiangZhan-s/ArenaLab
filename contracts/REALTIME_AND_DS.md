# 扩展规范 v1：实时模式与 Dedicated Server

本文件仅规范 `realtime` / `dedicated` 模式；`turn` 仍由 SEMANTICS.md 规范。不同模式必须有不同规则版本，不能共用“committed ACK”的含义。

## 实时最小玩法

二维 100×100 场地、2 名玩家、20 Hz 固定逻辑步长，每步最大位移 1 格。不实现枪械、碰撞物理、AOI 或技能。浏览器显示两个角色移动、tick、RTT、校正偏差。回合制棋盘仍保留。

输入为 session_epoch、seq、dx/dy，dx/dy ∈ {-1,0,1} 且 |dx|+|dy|≤1；不接受客户端位置、任意 dt 或终局胜者。每玩家每 tick 最多取一个合法输入；无输入则停止，不延续无限移动。接收序号必须严格递增，允许丢失后的缺口，不等待缺失输入；重复及倒序拒绝，未来序号跳跃有配置窗口。每玩家队列上限 128，超限拒绝并要求 resync。

初始 p0=(0,0)，p1=(99,99)，tick=0。快照包含 tick、位置、last_processed_seq、match/build 标识。快照 10 Hz，逻辑 20 Hz。客户端丢弃旧快照，安装权威状态后重放尚未确认的本地输入；校正偏差和未确认输入数量必须可观测。远端插值使用有界缓冲，不能称其为命中延迟补偿。

测试使用可控逻辑时钟；正常运行使用单调时间。严重停顿最多补 3 tick 后报告落后并重新调度，不无限追赶。记录 tick 执行时长与调度落后两个指标，不用执行耗时掩盖漏 tick。

输入响应 `accepted` 只表示入队；快照 last_processed_seq 表示内存处理完成。两者都不是持久化承诺。tick 内禁止同步数据库事务；数据库延迟不能直接卡住逻辑循环。

比赛固定 600 tick；演示规则为到场地中心目标点的 Manhattan 距离较小者获胜，相等平局。该规则只是可运行的同步实验，不包装成 FPS/MOBA。终局冻结一个 result_id，经持久 outbox 或重试协议投递结果；Go 后台提交结果和积分后返回 durable ACK。

实时进程在结果持久化前崩溃：默认 aborted/no-score，不承诺无损迁移；结果已提交但 ACK 丢失：重试原 result_id。恢复活跃战斗状态是另一个研究题，不自动套用 Lab07 回合制恢复。

## Dedicated Server 控制契约

本地进程池先支持两个以上独立 `ds-sim` 进程。状态：starting → ready → allocated → draining/stopped；失败进入 unhealthy，不再分配。一个实例同一时刻承载一场比赛。Ready 与进程存在不同，只有确认端口可服务且 build/region 匹配才分配。

allocation request_id 幂等，事务绑定 match、instance、generation；重试返回同一 allocation。容量耗尽返回 no_capacity，不创建幽灵占用。超时分配需对账，不能盲目换新 ID。

票据包含 player_id、match_id、allocation_id、generation、build、audience、exp、jti，并签名；DS 检查签名、目标、版本、过期和一次性消费。测试用服务端时钟；真实服务允许的时钟偏差须配置。票据不得出现在普通日志。重新接入发新票据，不能复用已消费 jti。

DS 用独立服务身份发送 heartbeat/result，玩家 token 不得使用此接口。result_id、allocation、generation、参赛者与预期胜者范围全部校验；同结果重试返回原结果，不同结果冲突隔离。信任 DS 结果不等于完成反作弊，失陷 DS 仍是信任边界。

租约过期/旧 generation 不得结算。结果提交与关闭 allocation 共用串行化边界；谁先提交决定允许终态。drain 停止接新局，现有局完成；超时强制退出必须明确 aborted，不能随意补一个胜者。

控制接口建议 HTTP JSON：register、ready、heartbeat、allocate、join-ticket、result、drain；客户端游戏流量直连 DS，不强制经过 Go 网关。Go ds-sim 与 UE 接入用同一控制协议，但游戏流量协议不同，禁止宣称兼容 UE Replication。
