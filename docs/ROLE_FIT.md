# 岗位适配边界：Go 游戏后台，不是假装战斗引擎

## 三层能力，分别交付

| 方向 | 本课程能提供的证据 | 不能由此声称 |
|---|---|---|
| FPS/MOBA 的 Go 在线后台 | 登录会话、组局、分配、build/region 隔离、接入票据、DS 结果鉴权、结算、故障隔离 | 完成完整 FPS/MOBA 战斗服、拥有生产用户 |
| 实时房间/同步岗位 | 20 Hz 权威二维移动实验、输入序列、客户端预测与校正、插值、tick 预算、慢连接和有界队列 | 完成射击命中回溯、物理预测、MOBA 确定性技能系统、大地图 AOI |
| UE 项目的后端集成 | Go 分配服务与独立 DS 的生命周期和协议；真实 UE 联调后可证明 UE 接入 | Go 模拟 DS 等于 UE Dedicated Server；掌握 UE Replication、GAS 或引擎源码 |

这是能力映射，不是腾讯、字节或任何部门的录用承诺。逐岗位看 JD：偏平台后台可主讲主线；偏战斗同步可主讲 Lab12，但还需专项训练；要求 C++/UE 核心开发的岗位不能靠这个项目完全适配。

## 真实需求与选择来源

玩家需要匹配到版本兼容、尚有容量的对局；服务器宕机不能产生两份结果；比赛服提交结果而非玩家自报胜负。这些是独立于游戏美术和引擎的后台问题。

UE 自己有权威服务端和复制网络。Go 在这里负责进入对局前、离开对局后及比赛服之间的协调，不替代 UE 的战斗网络。

主线先完成可玩的回合制闭环，便于验证持久化和事务；实时实验使用独立模式，不把回合制的逐行动持久化生搬到 tick。最终展示两条旅程：浏览器回合制；匹配 → 分配 Go DS 模拟进程 → 实时移动 → DS 上报结果 → 结算。

## 真实 UE 联调的额外门槛

Lab14 的 UE milestone 独立标记 `not_run / passed / failed`。需要可合法取得的 UE 源码与构建环境、可用 C++ 工具链，以及足够编译磁盘与时间。它不是 Go 主线启动的前置条件。

最低联调：使用官方专用服务器样例，两个客户端经 Go 获取 endpoint/ticket；DS 在接入层验证票据，报告 ready/health/result；Go 验证 DS 身份并结算。只允许少量 HTTP/接入胶水或 sidecar，不实现引擎。票据验证必须发生在授权加入比赛之前，不能只有 sidecar 自说成功。

保存 UE 版本/构建标识、服务器启动命令、客户端连接日志、失效票据拒绝、一次 DS 退出后的回收和积分证据。没有这些材料，简历只能写“定义并验证独立 DS 接口”，不能写“已集成 UE”。

## 一手资料与推导（核查于 2026-11-09）

- [Epic 网络概览](https://dev.epicgames.com/documentation/en-us/unreal-engine/networking-overview-for-unreal-engine)：UE 网络的客户端/权威服务器职责。由此划定 Go 控制服务与 UE 游戏状态复制边界。
- [Epic Dedicated Server 设置](https://dev.epicgames.com/documentation/en-us/unreal-engine/setting-up-dedicated-servers-in-unreal-engine)：实际 UE DS 构建和启动以所选版本官方步骤为准。
- [Agones 分配接口](https://agones.dev/site/docs/reference/gameserverallocation/)：参考可用实例选择与分配生命周期；课程先实现本地进程池，不要求 Kubernetes。
- [Agones SDK 生命周期](https://agones.dev/site/docs/guides/client-sdks/)：参考 Ready/Health/Shutdown 职责，不宣称自制协议兼容 Agones。
- [Nakama 多人游戏模型](https://heroiclabs.com/docs/nakama/concepts/multiplayer/)：对比权威、转发和房间模式。

以上是工程边界依据，不是这些产品背书课程或证明某公司必考的材料。
