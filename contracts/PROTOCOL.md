# 协议实现指南
本指南与SEMANTICS.md一起约束Lab02～08；JSON Schema仅验证结构，不能替代业务授权。
## HTTP（对应Lab才实现）
POST /v1/sessions：本地测试凭据→短期票据，禁止任意身份自注册。
POST /v1/tickets：已认证玩家排队；重复请求复用request_id。
GET /v1/tickets/{id}：只允许本人查询assignment与终态。
DELETE /v1/tickets/{id}：取消是状态竞争，不是物理删除票据。
GET /v1/matches/{id}：仅参与玩家可读快照/结果。
GET /v1/leaderboard：分页读取赛季投影并注明更新时间。
GET /healthz：进程存活。GET /readyz：能否接收新分配；drain时为false。
GET /v1/ws：升级连接，短期票据认证，Origin与大小限制。
## WS客户端消息
join：match_id与票据；服务端查成员。
ready：match_id；双方已ready的状态变更必须可重复。
action：见websocket.schema.json。没有player_id，身份来自会话。
resume：match_id、last_seen_version；成员认证后请求连续事件或快照。
heartbeat：连接活性，不能当业务提交确认。
## WS服务端消息
ack：command_id、status=committed、committed_version、command_result。Lab04前只能provisional。
reject：command_id、code、retryable、current_version；错误未知结果不得伪装业务拒绝。
snapshot：match_id、state_version、rules_version、完整状态。
event：match_id、state_version、command_id或timeout事件ID、事件类型、确定载荷。
terminal：match_id、result_id、final_version、rules_version、season_id、winner；能从持久化结果重新获取。
## 结果和状态区分
重放旧命令时ACK仍指向该命令的旧committed_version；不能因此把客户端最新快照回退。客户端需要当前状态时单独resume。
## 请求预算
在Lab02先定默认帧大小/连接空闲时限/玩家速率和有界队列容量，集中配置并记录实验环境。不要将任意数字当生产通用值。
## 序列化
状态hash使用固定字段顺序与编码，不直接hash无序map。记录规则版本；事件不能包含重放时才计算的随机数或当前时间。
