# ArenaLab 教学协作规则
本仓库包含课程维护与学习实践两类工作。明确要求完善课程时可以修改全部教学材料；学习实践遵守 Gate，不自动越级。
每次实践先读 PROGRESS.md、contracts/SEMANTICS.md、当前 Lab 的 README、tasks、acceptance.json 和 learning/labXX/prelab.md。
00～11 还须读 execution.md；12～14 读 contracts/REALTIME_AND_DS.md。固定场景在 fixtures/gates，执行契约在 docs/EXECUTABLE_GATES.md。课程维护不得代填模型答案或学习者预测。
初始预测为空时先提问。LEARNER-OWNED 段落只能由学习者填写。
每次只实现一个可验证任务；先说明改动文件、验收断言和禁止范围。
修改后交付：执行路径、事务边界、故障窗口、测试结果、3 道追问、剩余限制。
不得删除固定验收、放宽协议、用 skip 掩盖未实现、编造性能数据或将绿色 CI 等同掌握。
契约变更必须先独立记录 ADR，修改 fixtures 并解释旧断言为何不再成立。
关键边界：先持久化再 committed ACK；先鉴权再读取幂等结果；epoch 与状态提交在同一串行化边界检查。
课程工具与规则 stub 是支架；业务实现按 Lab 推进。终局与积分结算是不同事实。
不得把课程维护生成的文字写进学习者反思，不得把 mutation 演示标为生产故障测试。
driver不得接收期望值后伪造观察，必须操作真实连接/进程/数据库；不能持全局锁掩盖业务并发。通过固定场景与人工解释分别记录。
实时tick不沿用回合制durable ACK；Go DS模拟通过不得标为真实UE联调通过。故障控制仅test构建，reset只准独立测试命名空间。
