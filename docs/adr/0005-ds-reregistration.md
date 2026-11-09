# ADR 0005：空闲 DS 重新注册

实测 WSL 停止后重启演示，后台过期扫描先于 DS 注册运行，将尚未分配的实例标成 unhealthy。原先注册只更新心跳，导致空闲实例永远 no_capacity。

注册仅允许“从未有 allocation 的 unhealthy 身份”恢复 ready；已消费/活跃的 one-shot 身份不自动复用。它没有解决生产级实例 incarnation/generation 管理，自动回收仍是扩展任务。新增真实数据库回归检查该启动顺序。

同一固定配对的两个玩家可能同时请求 allocate，统一按 player ID 顺序锁定双方，避免两份独立请求分别持有一边的锁后争抢 occupancy。
