# 参考版本干净检出证据

被测代码：`0e5b8ef48d949dfdf1c42c0a9b34a10915c565ba`，干净 clone 到 WSL ext4；后续教学说明/证据提交不改变该被测实现。

命令：`bash scripts/clean-room.sh`。机器验收子命令：`bash scripts/wsl-reference.sh`。原始 run：`20261109T093938Z`。

环境：WSL2 Linux x86_64，Intel Core i9-14900HX，WSL 可见 32 逻辑 CPU、15849 MiB RAM；Go 1.25.14；Docker 29.1.3 / Compose 2.40.3；MySQL 8.4 系列、Redis 7.4；验证 MySQL 数据目录为 tmpfs。没有设置容器 CPU 限额，非独占生产压测环境。

## 固定机器验收

Lab00 参考模型、Lab01 规则 fixture、Lab02–14 共29场景全部通过。TestRealProcessGates 耗时178.12s，额外过期/票据/注册回归0.58s，整个 e2e 包179.718s。reference vet、Go race 及子进程日志 race/panic 检查通过。

原始日志：`artifacts/raw/runs/20261109T093938Z/gates.log`。
SHA256：`29df2a2376c63311052fb857bc46e49f5074c330523f49de407bbe966508c9f5`。

## 开放到达率基线（不是容量上限，也不是优化对照）

| 指标 | 本次观测 |
|---|---:|
| 活跃房间 | 100 |
| offered rate | 200 请求/秒 |
| 计划窗口 | 30 秒 |
| offered / succeeded | 6000 / 6000 |
| rejected / timeout / error | 0 / 0 / 0 |
| completed rate（含尾部排空） | 199.9795 请求/秒 |
| 全样本 P50 / P95 / P99 | 6.106 / 7.582 / 25.057 ms |

延迟从计划到达时刻计算，包含负载机排队。服务与驱动为 race 构建；tmpfs 数据库不能用于推断磁盘持久化性能，进程 kill 不能证明断电持久性。这里只是一轮教学 Gate 基线，未做三次 A/B，不写“提升了 X%”。

样本：`artifacts/raw/runs/20261109T093938Z/reference/open-loop-3949540855.json`。
SHA256：`4bef3bb837dc2ae308d224b1ea5bb52e2088c2cb24273d4a2d018a53b0a6913a`。
复算：`go run ./cmd/measure <该样本路径>`。

## 变异

五个变异均先成功编译，再被指定业务断言判失败：ack-without-commit、version-before-dedup、missing-fence、settlement-not-idempotent、expired-ticket-accepted。不是靠编译错误计为 killed；判卷器和固定期望未修改。日志在 `artifacts/raw/mutations/`。

## 空库、浏览器与核账

完整 clean-room 脚本退出码 **0**。从已提交源码检出，不复制旧工作区的 bin、node_modules、learning 产物或数据库；Docker 镜像与 Go 编译缓存可复用，因此不声称“断网且所有镜像缓存为空”的验证。

新 Compose 项目 `arenalab-clean-20261109094411` 创建新 MySQL 卷，7 个服务启动。Playwright Docker 入口真实执行，2026-11-09 09:44:52–09:45:25 UTC：

- turn：两个独立浏览器 context，重连后均 version41，同 match/终局。
- dedicated：两个独立浏览器 context，移动与重连后均 tick600，同 match/终局。
- pageErrors 空；每局 MySQL 恰有1个 result、2条 ledger，state=applied，积分与胜者对应。

浏览器 report SHA256：`446b1cea017a9a932d1865c576eb6955b065bdb710a3378fbffd0177e127bc10`。
SQL 核账 SHA256：`73aa08848b969d775eeeb41cc00b7f283cf558fc58e6432024884833f9a3be1f`。

本机原始归档：`artifacts/raw/arenalab-clean-20261109094411.tar.gz`，19395299 bytes。
SHA256：`862ce7a5439ffab6d7fb8860060a03df5c01dfa05f83e7e2c27b89e2bd0ab5fb`。
原始归档不入 Git；可在本机解压核验。仓库上传可复现代码与摘要，GitHub Actions 配置为每次运行另行上传该次证据，未将本地成功冒充远端 CI 成功。

额外修复：空闲 DS 注册恢复顺序见 ADR0005；容器产物在 ext4 上原属 root，脚本已将证据所有权归还调用者。本轮实测 reference 证据目录属 wuyi、0700，可由学习者读取。
