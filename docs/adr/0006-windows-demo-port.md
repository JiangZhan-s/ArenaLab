# ADR 0006：Windows 宿主机演示入口

2026-11-09 实测 WSL 内 8080 返回健康状态，Windows 拒绝连接。`netsh interface ipv4 show excludedportrange protocol=tcp` 显示系统保留 7998–8097，8080 在其中；8101–8103 则已由 wslrelay 监听。

本地 Compose 演示改为 18080，同时修改 arena 健康检查、DS 后台地址、浏览器测试入口和 Origin 白名单。DS 的 Origin 由启动配置传入，仍精确匹配；不改成允许所有来源。独立测试默认 8080 Origin 保持兼容，测试服务实际使用动态端口。

不删除系统保留区间、不修改防火墙或全局 WSL 网络模式。以后排障必须同时检查 WSL 服务健康与 Windows 宿主机实际连通性，WSL 内 Chromium 通过不能替代宿主机验证。

验证（实现提交 86878be）：Windows `curl.exe --noproxy "*" http://localhost:18080/healthz` 返回200；主页与 app.js 返回200，Windows 直连8101健康检查成功，18080由wslrelay监听。相关 Go race 单测通过；WSL 的双浏览器回合制与 DS 旅程及独立 SQL 核账全部通过。Windows验证为宿主机HTTP/资源连通性，不冒充Windows GUI浏览器自动化。
