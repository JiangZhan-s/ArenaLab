# 规范性业务契约 v1
本文件是规范来源；历史规划用于解释动机。发现矛盾先修契约，不自行选有利于实现的版本。
## 规则
5×5，坐标 x/y ∈ [0,4]。玩家槽位 0、1，初始 (0,0)/(4,4)，HP 3。turn 是当前槽位，ply 是已完成行动数，初始 0；version 初始 1。
每次 move/attack/pass/timeout 成功都使 ply、version 加一。未终局时 turn 翻转；终局保留本次行动者的 turn。
move 的 dx/dy 必须满足 |dx|+|dy|=1，不能出界/与对手重合。attack 只能攻击正交相邻敌人，伤害 1。pass 与 attack 的 dx/dy 必须为 0。
HP=0 优先触发终局；否则第40次行动后按 HP 判胜；相等平局。winner 为 0/1，平局或未终局为 -1。
纯规则函数只支持 move/attack/pass；timeout 在服务端被授权后转成 pass，同时持久化事件原因 timeout，客户端无权直接提交 timeout。
非法操作不修改任何字段，返回原状态。拒绝错误顺序：finished、not_turn、invalid_action、illegal_move 或 not_adjacent。
## ACK 与错误
Lab 01～03 仅内存 provisional 状态，不得声称 committed。Lab 04 开始：认证 → 授权 → 当前会话校验 → 幂等查询 → 锁定并验证当前版本/截止时间 → 写状态/事件/结果 → commit → 更新内存 → ACK/广播。
同 command_id 同语义 payload：重放原已提交结果，即使 expected_version 已过时。不同 payload：idempotency_conflict。非已提交拒绝可重新验证，不预留成功。
payload hash 包含协议、对局、动作、expected_version；排除 session_epoch，使换连接后可用新会话重试原命令。旧会话即使查旧结果也先拒绝。
commit 返回连接错误时结果不确定：不得立即用新 ID重做；查询原 command_id 确定结果。应用错误与结果未知分开。
## 会话与连接
dev 登录仅使用服务端配置的测试玩家凭据，不允许任意 player_id 直接登录。短期票据绑定身份、expiry、session_epoch。
认证后替换连接提升 epoch。Lab 02 单进程，Lab 07 前补持久化/跨节点会话串行化；session epoch 的替换与行动提交按 player 行锁串行化。替换前已提交命令仍有效，替换后旧会话不能写。
WebSocket origin 白名单、帧大小上限、读写 deadline、每玩家请求预算、退出关闭顺序必须在 Lab 02/03 实验中记录。
## 状态机
Ticket queued → assigned/cancelled/expired，无终态回退。
Match allocated → waiting_ready → running → finished；allocated/waiting_ready/running 可因明确原因 → aborted。终态不再行动。
Connection authenticated → attached → replaced/closed。
Settlement pending → applied/quarantined；quarantined 经人工审计可重新 pending，不创建第二个积分流水。
ready 截止时间由配置固定并持久化；不齐则 aborted，无积分。断线不暂停进行中回合。
## 时间、恢复与 owner
事务获取相关锁后读取数据库当前时间再比较 deadline；不要使用锁等待前缓存的时间。now < deadline 接受，等于/超过则拒绝行动并由 timeout 流程推进。
房间停顿恢复后只生成当前回合一次 timeout，新 deadline 从处理时刻 +15s。
Lab 07 先比较 abort/no-score 与恢复两种策略；选择恢复分支才引入 owner epoch。有效租约/epoch 与状态版本在同一事务串行化检查。
建议锁序：会话 player 行按 ID → owner 行 → match 行；匹配：player occupancy 按 ID → tickets 按 ID → assignment。结算：job → ratings 按 player ID。跨流程扩大锁集合前必须更新锁序 ADR。
一局事件顺序使用 state_version；owner_epoch 不替代 state_version。新 owner commit 后旧广播可能延迟到达，客户端以版本去重/检测缺口，不能简单以 epoch 丢弃必要历史事件。
## 重连
last_seen_version 后连续事件可补发；缺口或清理后返回快照。快照必须标注 version，安装后只接受连续增量。
状态广播只有完整快照可覆盖合并；增量不可随意丢弃。ACK/终局队列满则断连，通过幂等查询或重连恢复。
## 结算
规则版本、赛季在开局时固定。胜+10、负-10、平0，允许负分，无下限截断。初始积分0，汇总=流水之和。
(match_id,player_id) 唯一；同一结果重放幂等；不同结果冲突隔离告警。aborted 无计分。终局、result、pending job 同事务。
Redis 是投影，使用带版本的绝对积分写入；跨 key 的版本检查与更新须原子。不得先 INCR 再另记消费完成。
