# 文件职责
contracts/SEMANTICS.md：唯一业务语义来源。
contracts/PROTOCOL.md：HTTP/WS消息合同；Schema只覆盖action结构。
contracts/invariants.json：I01～I12编号。
ACCEPTANCE.md：G01～G18故障定义；这里G表示故障，不是课程毕业状态。
course.json：Lab清单和依赖；每Lab acceptance.json是测试计划，status初始not_implemented。
internal/rules：Lab01接口、默认 exercise stub、reference 标签启用的参考答案。
test/acceptance/rules：固定验收与少量错误候选检测；acceptance构建标签才启用业务实现测试。
cmd/course：课程结构检查与gate入口。02～14固定 runner 位于 internal/coursegate，真实参考驱动位于 internal/reference/testdriver。
learning：学习者证据；templates：可复制模板；prompts：不同协作角色。
docs/COURSE_STATUS.md：课程本身的交付边界。
历史规划中YAML蓝图是建议布局；当前选择JSON来用Go标准库验证，course.json与实际目录为准。
# 新版增量导航

15 Lab主线以course.json为准；00～11另读execution.md，12～14为实时/独立DS扩展。
岗位边界：docs/ROLE_FIT.md；扩展规范：contracts/REALTIME_AND_DS.md。
固定驱动场景：fixtures/gates；运行器：internal/coursegate；量化工具：cmd/measure与internal/measurement。
运行说明：docs/EXECUTABLE_GATES.md；交付合同：docs/FINAL_DELIVERY.md；工作量与面试证据：docs/MEASUREMENT_AND_INTERVIEW.md。

reference/model.json：维护者模型答案，不写入 learning。internal/reference：SQL/服务/worker/实时/DS/真实客户端与驱动。db/migrations：嵌入式建表和兼容迁移。web：两模式浏览器 UI。
cmd/arena-server、settlement-worker、ds-sim：真实进程；cmd/demo-check：浏览器结果独立 SQL 核账；cmd/mutation：隔离源码副本的变异运行器。
scripts/wsl-reference.sh、wsl-browser.sh、wsl-mutations.sh：维护者自动验证入口。docs/REFERENCE_GUIDE.md：学习使用方式；docs/VALIDATION.md：证据；docs/COVERAGE.md：未覆盖项。

