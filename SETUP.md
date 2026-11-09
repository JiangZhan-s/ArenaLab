# WSL 环境与命令

推荐在 WSL 终端运行全部命令，并将 clone 放在 Linux 文件系统，例如 `~/src/ArenaLab`。不要在 Windows PowerShell 直接执行 Linux 脚本。Docker Desktop 的 WSL integration 或 WSL 内 Docker Engine 二选一，先确认 `docker info` 和 `docker compose version` 成功。

本次验证：WSL Linux x86_64、Ubuntu 24.04、Docker 29.1.3、Compose 2.40.3；Go 工具链使用 `golang:1.25` 容器。数据库 MySQL 8.4 / Redis 7.4，浏览器 Playwright 1.55.1。版本标签不等于不可变摘要，实验时记录实际版本与 image digest。

WSL 内原生 Docker：保持一个 WSL 终端打开。仅从 Windows 短暂执行 `wsl -- command`，全部前台进程退出后，本机实测 WSL 会自动停止，systemd/Docker 本身不保证保活。可在一个终端运行 `bash scripts/demo.sh --attach` 持续查看日志，在第二个 WSL 终端执行浏览器测试。退出日志不会删除数据；重新进入 WSL 后再执行 demo 启动即可。

```bash
bash scripts/demo.sh
bash scripts/wsl-reference.sh
bash scripts/wsl-reference.sh 04,07
bash scripts/wsl-browser.sh
```

演示入口 http://localhost:8080，DS 使用 8101–8103；演示数据库 23306/26379，验证数据库 13306/16379，均仅绑定 loopback。端口已占用时先查明占用者，不杀未知进程。首次运行需要下载镜像；仓库 vendor 固定 Go 依赖，不依赖运行时访问 Go module proxy。

演示数据使用 `arenalab-demo_demo-mysql` 卷；`docker compose -f deploy/compose.yaml stop` 保留它。`down -v` 会不可恢复地删除本地演示数据，日常学习不要使用。验证库是 tmpfs，停止数据库容器会丢失测试数据。driver 只创建随机 `arenalab_test_` 数据库，不使用演示库，不自动删除其他数据库。

本机已经遇到 `/mnt/c` 上 npm 安装文件损坏：优先在 ext4 clone。浏览器 Docker 脚本用独立 node_modules 卷避开此问题。若 mcr 镜像下载不可达，可使用 `bash scripts/wsl-native-browser.sh`，它需要 WSL 自己的 Node/npm 和 Chromium 系统依赖，并将依赖安装至 `/tmp/arenalab-browser.*`；缺少依赖时报告错误，不把未运行记成通过。

学习时若希望直接使用 Go，安装 Go 1.25+ 后运行：

```bash
go run ./cmd/course check
go test ./...
go vet ./...
go run ./cmd/course gate 01  # 初始红灯：学习版规则尚未实现
go run -tags reference ./cmd/course gate 01
```

`-race` 需要 Linux C 工具链；reference 容器已经具备。详情见 [参考手册](docs/REFERENCE_GUIDE.md)。
