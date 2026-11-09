# Lab 04 分步任务

## T1：创建MySQL迁移和存储接口，提交状态/事件/命令结果同事务

开工前：写允许/禁止结果与最小反例。
实现后：提交可审查diff，保存运行命令和观测证据。失败时先写两个根因假设，再请求修复。

## T2：明确幂等hash与重连epoch关系，成功重试先于过期version拒绝

开工前：写允许/禁止结果与最小反例。
实现后：提交可审查diff，保存运行命令和观测证据。失败时先写两个根因假设，再请求修复。

## T3：把ACK移到commit后；处理commit结果未知，按command_id查询

开工前：写允许/禁止结果与最小反例。
实现后：提交可审查diff，保存运行命令和观测证据。失败时先写两个根因假设，再请求修复。

## T4：实现子进程控制点，分别在commit前和commit后ACK前kill

开工前：写允许/禁止结果与最小反例。
实现后：提交可审查diff，保存运行命令和观测证据。失败时先写两个根因假设，再请求修复。

## 指定故障
同command20次，异payload复用ID；kill后重启并查询持久化事实。

## 文件范围
- `db/migrations/000001_match.sql`
- `internal/persistence/`
- `test/integration/command_test.go`
- `test/process/action_crash_test.go`
这些是本Lab目标交付物，初始可能尚不存在。固定验收先提交，再实现业务。

## 深入理解
连接断开时可能已提交；应以同业务ID查询，不能立即用新ID重做。
