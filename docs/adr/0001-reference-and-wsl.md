# ADR 0001：参考实现隔离与 WSL 验证

状态：实施中，2026-11-09。

用户要求完整参考实现并指定WSL。练习默认入口保留stub，`-tags reference`显式选择教师实现；参考模型答案独立存放，不写入learning。参考通过不修改学习者PROGRESS。

现有验收遗漏：gate00硬编码读取学习者文件，无法区分教师验证；gate01子进程丢失编译标签。增加reference模式选择，保持默认练习失败语义。没有放宽业务断言。

环境：Ubuntu-22.04发行版中已有Docker Engine；Windows无Docker命令不代表WSL无Docker。以WSL中的容器和真实进程验证SQL、网络及故障；不得用内存存储替代持久化Gate。

参考WS使用github.com/gorilla/websocket v1.5.3，固定版本以复现；每连接一个reader和一个writer，队列有界。数据库使用go-sql-driver/mysql v1.9.3。发布前需额外依赖漏洞扫描，不因版本固定就声称安全。

官方API依据：https://pkg.go.dev/github.com/gorilla/websocket@v1.5.3 与 https://pkg.go.dev/github.com/go-sql-driver/mysql@v1.9.3。
