# 第一次学习
## 第一次 60～90 分钟
1. 阅读 PROJECT_AND_COURSE.md 的玩法、玩家旅程和业务承诺。
2. 阅读 contracts/SEMANTICS.md。用自己的话解释行动成功与收到消息的差别。
3. 填写 learning/lab00/prelab.md，保留错误预测。
4. 将 prompts/session.md 发给 Codex，开始一次只问一题的质询。
5. 完成 Lab 00 的四张状态表和十个异常事件序列，才进入 Lab 01。
## 环境
安装可用的 Go 工具链（模块最低 Go 1.25），Git；Docker 到 Lab 04 再使用。
在仓库根目录执行：
```text
go run ./cmd/course check
go test ./...
go run ./cmd/course gate 01
```
前两条检查支架；第三条运行 Lab 01 规则验收，初始应失败并提示未实现。
Lab 00 先按 templates/model.json 自行填写 learning/lab00/model.json，再运行 gate 00；人工状态表和答辩仍必须完成。
Lab 02～14 的固定场景与验收器已提供；实际业务和测试驱动逐 Lab 实现。启动方式、隔离要求见 docs/EXECUTABLE_GATES.md。不要将缺驱动的连接失败记为通过。
00～11 开始前另读该 Lab 的 execution.md；12～14 另读 contracts/REALTIME_AND_DS.md。最终完成条件见 docs/FINAL_DELIVERY.md。
当前支架不需要数据库、模型 API key 或付费服务。
## 恢复学习
新会话让 Codex 阅读 AGENTS.md、PROGRESS.md 和当前 prelab。每次结束将实际测试命令和 SHA 写入学习证据，更新进度。Git 分支使用 lab/01-rules 等名称。
