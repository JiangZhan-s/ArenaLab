# 分步任务

## T1：独立进程与ready池

实现：启动至少2个独立ds-sim进程，注册build/region/endpoint。实现ready、health、drain；验证进程存在不代表可分配。

验证与学习：对比按需启动与预热，测冷启动和分配P99，保存进程ID。

开始前由学习者写预测；完成后交付代码路径、反例、原始输出。

## T2：幂等分配

实现：request_id去重，事务绑定instance/generation/match。并发分配不能双占，no_capacity不留占用。

验证与学习：在提交后响应前断连接，原request_id重试应返回同一assignment。

开始前由学习者写预测；完成后交付代码路径、反例、原始输出。

## T3：接入授权

实现：签发短期一次性票据；DS验证audience/build/generation/expiry/jti；重连发新票据。

验证与学习：篡改、过期、重放和跨局票据均拒绝；日志不能输出票据。

开始前由学习者写预测；完成后交付代码路径、反例、原始输出。

## T4：可信结果与实例故障

实现：DS服务身份和玩家身份分离；终局结果幂等，过期generation拒绝；kill旧DS后不得结算。

验证与学习：运行gate13，移除鉴权和generation检查分别证明测试变红。

开始前由学习者写预测；完成后交付代码路径、反例、原始输出。
