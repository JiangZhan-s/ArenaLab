# 最终可运行交付合同

这是课程完成标准，不是当前已经可运行的声明。当前状态见 COURSE_STATUS.md。

## 必须实现的产物

| 产物 | 最迟阶段 | 完成定义 |
|---|---|---|
| cmd/arena-server + web/turn | Lab11 | 双浏览器可玩，非仅curl返回200 |
| cmd/testdriver | Lab02开始逐层扩展 | 操作真实客户端和子进程；固定Gate无需修改期望即可执行 |
| migrations + settlement-worker | Lab08 | 空库可初始化，结果和流水可独立核账 |
| loadtest + 原始采样 | Lab09 | 开放到达率、失败/超时完整记录，汇总可复算 |
| web/realtime + internal/realtime | Lab12 | 双客户端可移动、校正可见、20Hz逻辑无逐tick SQL |
| cmd/ds-sim | Lab13 | 独立进程，ready/health/allocate/ticket/result链路真实 |
| deploy/compose.yaml + demo脚本 | Lab14 | 新环境按文档启动，无手改数据库、隐藏本地依赖 |
| integration/unreal | 独立里程碑 | 真UE DS双客户端、票据拒绝、结果和进程故障证据 |

## 最终启动接口（待各 Lab 实现后才可用）

固定交付命令：`docker compose -f deploy/compose.yaml up --build -d --wait`。
容器包含MySQL、Redis、arena、worker、至少两个ds-sim；浏览器入口 `http://localhost:8080`。正常profile不启动测试驱动。

Gate profile：`docker compose -f deploy/compose.yaml --profile test up --build -d --wait`，驱动仅发布127.0.0.1:8099，单独测试数据库，不影响演示对局。需要Docker Compose支持--wait；SETUP.md在实现时固定验证过的版本。

Windows提供 scripts/demo.ps1，Linux提供 scripts/demo.sh；二者分别启动、检查就绪、打印URL并给出退出命令。不要复制依赖Linux shell的命令假装Windows也已测试。

最终验证依次执行：

```text
go test ./...
go test -race ./...
go run ./cmd/course check
go run ./cmd/course gate-all
```

Windows无race工具链时使用Linux CI完成，必须保存对应SHA结果。starter CI不是以上全链路证据。

## 两条演示必须亲手走通

1. 两个浏览器登录 → 匹配 → 回合制行动 → 关闭一个窗口 → 重连 → 结束 → 查询一致状态与积分。另开对局，在ACK前后kill，解释哪个承诺成立。
2. 两个浏览器 → 匹配与DS分配 → 票据接入 → 实时移动 → 查看预测校正 → DS终局上报 → 结算；重试结果不重复计分，杀DS未完成局不计分。

最终验收中既有固定bot journey，又有人看浏览器交互；driver里的journey通过不证明HTML真的可用。演示录屏需标SHA、时间和启动环境，不能用静态截图代替。

## 完成矩阵

docs/evidence/index.md逐项填写：SHA、命令、环境、exit code、原始文件、可解释边界、复现者。保留not_run/failed/passed，不能空着默认passed。

- [ ] 00～14机器验收逐项通过；原故障清单逐项映射测试或明确未覆盖。
- [ ] 每个Lab人工review与迁移答辩通过。
- [ ] 全新空库启动与两条浏览器旅程通过。
- [ ] 五个关键故障主题至少各一个mutation证据。
- [ ] 至少三组对照实验，各方案至少三次原始记录。
- [ ] release无测试故障端点，无硬编码生产凭据。
- [ ] 另一人按README成功复现；受限环境明确写限制。
- [ ] UE真实集成单独记录；未通过则收窄简历措辞，不阻止Go主线学习。

没有这些证据只能称“进行中的课程项目”。即使全部完成，生产上线仍需安全审计、真实流量容量验证、运营和长期故障经验，不能自动称为生产级。
