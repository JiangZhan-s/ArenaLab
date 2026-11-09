# 表设计审查指南
## Lab04
matches(match_id PK, status, rules_version, season_id, ready_deadline)。
match_players(match_id, player_id UNIQUE组合, slot)，每局slot唯一。
match_states(match_id PK, state_version, turn, ply, state_payload, deadline)。
match_events(match_id,state_version UNIQUE, event_id UNIQUE, type,payload,occurred_at)。
command_results(match_id,player_id,command_id UNIQUE, payload_hash,committed_version,result)。
sessions(player_id PK,epoch,expires_at,credential_reference)。不存明文登录凭据。
## Lab05
player_occupancy(player_id PK,kind,ticket_id,match_id,version)保护唯一占用。
tickets(ticket_id PK,player_id,status,requested_at,rating_snapshot,region,version)。
assignments(ticket_id UNIQUE,match_id,server_id)。
占用、票据终态、match及assignment以事务关联；按player_id排序取锁。
## Lab07
room_owners(match_id PK,owner_id,epoch,lease_until)。更新lease不可让旧epoch续租。
snapshots(match_id,version UNIQUE,rules_version,payload,checksum)。
事件裁剪前验证快照可恢复、重连有快照退路并保留足够审计依据。
## Lab08
match_results(match_id UNIQUE,result_id UNIQUE,winner,rules_version,season_id,final_version)。
settlement_jobs(match_id UNIQUE,status,attempts,next_attempt_at,lease_until,owner)。
rating_ledger(match_id,player_id UNIQUE,season_id,delta,result_id)。
player_ratings(season_id,player_id UNIQUE,rating,projection_version)。
两人流水、两人汇总及job完成同事务，按玩家ID排序锁。
## 跨阶段要求
索引必须能对应查询；唯一约束以数据库为准；JSON快照不替代可查询关键列。
CHECK保护位置/非负HP等局部约束；跨行积分与玩家数量由事务和验收保护。
所有时间UTC、精度写入规范；ID不可被当成权限判断。
