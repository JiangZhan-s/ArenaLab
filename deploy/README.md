# WSL 部署

`compose.yaml` 是本地参考演示，`compose.verify.yaml` 是独立验收基础设施。入口分别是 `bash scripts/demo.sh` 和 `bash scripts/wsl-reference.sh`，不要用验证库当长期开发数据库。

演示 MySQL 使用持久卷；验证 MySQL 使用 tmpfs 和随机数据库。全部端口只在本地回环接口使用。Compose 内明文凭据是公开的教学口令，禁止复用或暴露到公网；生产部署必须另做身份、TLS、权限、密钥与备份设计。

镜像是静态 Go 非 root 运行时，不包含故障控制构建标签。DS 与后台使用 Linux host networking，因此推荐 WSL Linux Docker，不能声称未经验证的 Windows 原生容器兼容。

Lab10 是一致性逻辑备份恢复到新测试库并启动新实例，不是 binlog PITR 或生产备份演练。参考实现最终形态使用 MySQL；学习路线仍允许 Lab02/03 先内存 provisional，再在 Lab04 建立 durable ACK。
