# 环境与命令
Go 最低1.25；实际安装时选择官方仍受支持的版本，记录 go version。当前模块无外部依赖。
```text
go run ./cmd/course check
go test ./...
go vet ./...
go run ./cmd/course gate 01
go test -race ./...
```
Windows race 可能需要本地 C 工具链，可使用 Linux CI。不能把未执行报告成通过。
Lab 04：Docker + MySQL，参考 deploy/README.md。实验都在独立测试库运行，不连接个人真实数据库。
逐 Lab 的 test 路径在 tasks.md；尚未存在的测试是任务交付物，不是现成实现。
将 OS、CPU、内存、Go、数据库版本、持久化参数、Git SHA 写入 docs/environment.md。
