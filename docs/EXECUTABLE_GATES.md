# 可执行验收：有测试程序不等于业务已通过

## 立即可执行

从仓库根目录：`go run ./cmd/course check` 检查 15 Lab 和固定场景格式；`go test ./...` 测验收器自身。这两项不认证游戏服务。
`go run ./cmd/course gate-all` 顺序运行00～14，收集所有失败并以非零退出；不会因某一Lab没实现而跳过后续Lab。

- `go run ./cmd/course gate 00`：读取学习者自己填写的 `learning/lab00/model.json`，对六个故障语义作机器检查；模板在 templates/model.json。人工状态表与答辩仍必需。
- `go run ./cmd/course gate 01`：执行已有纯规则固定验收，业务 stub 初始失败。
- `go run ./cmd/course gate 02` 至 `14`：执行 fixtures/gates/labNN.json 中固定序列，逐项比较真实观测。缺少驱动或服务时非零退出，不跳过。

PowerShell 设置：

```powershell
$env:ARENALAB_DRIVER_URL = 'http://127.0.0.1:8099'
$env:ARENALAB_TEST_TOKEN = 'local-disposable-test-token'
go run ./cmd/course gate 04
```

参考实现已经包含独立驱动，使用 `bash scripts/wsl-reference.sh` 自动编译服务、启动测试驱动并运行固定场景；部分 Lab 使用 `bash scripts/wsl-reference.sh 04,07`。CLI 手动模式须 `go run -tags reference ./cmd/testdriver -help` 查看参数。学习实现与参考实现的验收证据必须分开，连接失败不能算通过。

## 驱动职责：只操作和观测，不出判卷结果

驱动仅绑定 loopback，校验 Bearer token，以 `POST /execute` 接收 `{op,args}`，返回 JSON 事实对象。验收器不会发送 want/bounds。成功执行一次操作返回 HTTP 200；业务拒绝放 error 字段；驱动内部错误/未知操作返回非 200。不能把未知操作默认为成功。

固定 want 是对象子集匹配；数组完全相等；bounds 对顶层数值作闭区间检查，缺字段或错误类型失败。并发块同时发请求并等待全部结束。驱动必须允许同一场景并发操作，不能持有覆盖整个 /execute 的全局互斥锁，否则会隐藏业务竞争。

服务业务接口独立于驱动协议。driver 可以适配接口命名，但不得实现第二套规则、模拟正确结果或读取期望 fixture 拼回复。为被测服务编写 driver 是有边界的工程任务，不是随意重写验收。

## 操作绑定表

| 操作 | 必须实际执行的动作 / 事实来源 |
|---|---|
| reset | 清理独立测试命名空间；初始化给定 profile/seed；返回实际初始状态。严禁默认开发/生产数据库 |
| connect/send_frame/action/input | 用真实客户端协议；handle 保存真实连接；未指定 handle 时给指定测试玩家登录并复用连接 |
| observe(room/client/snapshot) | 读取真实服务状态或客户端收到的消息；不得重演规则推导结果 |
| observe(sql/restored_sql) | 独立只读 SQL 连接读取状态、事件、命令、占用、流水；不能相信服务汇总响应代替核账 |
| enqueue/cancel/allocate | 调用真实匹配/分配接口；返回业务响应 |
| lookup/resync | 用当前合法会话查询幂等结果或接收快照；更新驱动的客户端状态 |
| settle/projection | 触发真实 worker 重试/投影流程，随后查询 SQL/Redis；重复消费真实消息 |
| fault | 装载一次性屏障/故障；after_commit 与 before_commit 必须由服务在准确位置通知外部驱动，然后外部 kill |
| process | 外部启动、终止、暂停/恢复实际子进程；记录 PID/时间/退出码。暂停实验推荐 Linux 容器 STOP/CONT |
| await | 有截止时间地轮询真实条件；不能固定 sleep 后回 true。测试租约须小于指定等待窗口 |
| backup | 真实备份和恢复至另一测试数据库；read restored_sql 必须走新实例 |
| exercise | 操作多个真实连接/进程，按 args 执行实验；原始采样落盘再计算计数/分位数，不接受预制汇总 |
| journey | 两个真实协议客户端完成登录、匹配、对战、重连、结算；不用写 SQL 直接制造终局 |
| submit_result/ds_result | 分别以玩家/DS 身份调用真实结果接口；检查真实身份边界 |
| ds_join | 从 Go 取得票据，再连独立 DS 验证；replay 重用上次 jti，expired 使用测试时钟产生过期票据 |
| clock | 仅测试模式推进可注入时钟，不能让公网客户推进比赛时间 |

## 初始化与输出约定

`turn` 默认 running：p0/p1 按 SEMANTICS.md，version=1，零事件；默认 actions 使用当前合法会话，未 connect 的玩家才自动登录。epoch 初始0，connect 后1，替换后2。reset 不自动预连。

`empty` 无 ticket/match，占用0；`finished_p0` 为通过正常规则到达终局并持久化 result r1 的夹具，不提前结算；`realtime/running` 按 REALTIME_AND_DS 初始状态；`dedicated/ready` 启动真实空闲 ds-sim，build=ds-v1/region=local。首次 allocation 使用测试 ID a1/generation1；正式系统 ID 格式不受此限制，driver 可映射 ID，但不能映射事实值。

`classified:true` 表示驱动识别并保留了真实成功或协议定义的拒绝，连接错误、解析失败、未知错误不能归类成功。并发之后独立读取版本/占用，才判断恰好一次效果。

`events/commands` 为当前 match 的持久记录数，不计初始状态；`ledger_rows` 为当前测试比赛积分流水数；`client_versions` 为客户端真实安装版本；`tick_sql_commits` 仅计算 tick 路径 SQL commit，不包括启动/注册；goroutine_delta 为 churn 前后 GC/静置后的差，不能含 driver。

action中的owner_epoch用于driver选择旧owner进程/连接，不是让公网客户端指定可信owner；服务必须使用自身持有的epoch执行事务检查。Lab04重启需从数据库重载单进程状态，Lab07再增加跨节点选主与fencing。

clock/advance_ticks场景用手动测试时钟；exercise/tick_budget必须切回真实单调时钟，不能瞬间推进200个逻辑tick伪造耗时。open_loop的rooms表示并发活跃房间数；回合制终局后按正常流程补充新局，不能在一局最多40步的限制下把终局拒绝算成功。负载计划、补局耗时和失败都保留。

## 防止伪绿与测试局限

验收器只证明固定场景，不能证明任意并发交错。driver 是可信计算基的一部分，必须审查它调用的真实路径。每个关键 Gate 还需做 mutation：去掉幂等唯一键/owner 条件/票据检查等，确认对应场景变红，再恢复。

至少 Gate04 的 commit 前后杀进程、Gate07 双 owner、Gate08 重复结算、Gate13 票据和结果鉴权各保留一份 mutation 证据。禁止用 driver 内存的计数器代替 SQL 真值。Lab03 需 Go race 检查；race 不通过不能由 HTTP 场景通过抵消。

故障接口仅在显式测试构建中编译，并且独立端口；release 构建不存在 /execute 和故障端点。reset 只允许带 `arenalab_test_` 前缀的数据库及随机 run_id，验证后才能清理，默认拒绝。驱动记录 SHA、配置、operation 时间线和原始采样到 artifacts/raw/run_id，日志脱敏。

数值阈值是本课程本地演示预算，不是生产 SLA。资源不足可先记录失败；调整预算需要提前 ADR，并保留原始失败，不得事后改 fixture 冒充通过。
