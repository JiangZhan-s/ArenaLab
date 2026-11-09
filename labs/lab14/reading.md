# 阅读路线

先读仓库 contracts/REALTIME_AND_DS.md、docs/ROLE_FIT.md 和 docs/EXECUTABLE_GATES.md。

1. [Epic Networking Overview](https://dev.epicgames.com/documentation/en-us/unreal-engine/networking-overview-for-unreal-engine)：标出UE持有游戏权威状态的位置，不把Go控制接口当游戏复制协议。
2. [Agones SDK](https://agones.dev/site/docs/guides/client-sdks/)：比较ready、health、shutdown生命周期；不要求部署Agones。
3. [Nakama多人模型](https://heroiclabs.com/docs/nakama/concepts/multiplayer/)：区分权威裁决与纯转发。
4. [Go诊断](https://go.dev/doc/diagnostics)：选择能验证本Lab假设的profile，不只截图。

阅读产出：画一条输入/分配/结果路径，标出信任与持久化边界，写出一个官方产品不替你解决的问题。
