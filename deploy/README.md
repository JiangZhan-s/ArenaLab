# 基础设施阶段
Lab 00～03 只需Go。Lab 04 增加MySQL，Lab 07运行两个应用进程，Lab 08加入Redis投影。
数据库使用独立测试库、UTC；记录隔离级别、flush/binlog配置。驱动需固定版本与go.sum。
创建 compose.mysql.yaml 时仅绑定127.0.0.1、凭据来自未提交的.env。停止保留卷；重建测试数据必须限定具体测试库。
Lab 10演练备份到新实例恢复，不能仅确认备份文件存在。
