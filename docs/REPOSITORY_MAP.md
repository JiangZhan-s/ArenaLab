# 文件职责
contracts/SEMANTICS.md：唯一业务语义来源。
contracts/PROTOCOL.md：HTTP/WS消息合同；Schema只覆盖action结构。
contracts/invariants.json：I01～I12编号。
ACCEPTANCE.md：G01～G18故障定义；这里G表示故障，不是课程毕业状态。
course.json：Lab清单和依赖；每Lab acceptance.json是测试计划，status初始not_implemented。
internal/rules：Lab01接口和stub，不是参考答案。
test/acceptance/rules：固定验收与少量错误候选检测；acceptance构建标签才启用业务实现测试。
cmd/course：课程结构检查与gate入口。02～11的runner需要随Lab建设，当前明确失败。
learning：学习者证据；templates：可复制模板；prompts：不同协作角色。
docs/COURSE_STATUS.md：课程本身的交付边界。
历史规划中YAML蓝图是建议布局；当前选择JSON来用Go标准库验证，course.json与实际目录为准。
