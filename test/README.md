# 验收层次
starter：工具与fixture结构；规则 gate：固定输入/输出；integration：真实MySQL；process：真子进程kill；load：容量与恢复；defense：学习者答辩。
当前提供 Lab01 的可执行固定规则验收；业务测试G02～G18由对应Lab创建并审查，未实现状态必须显式失败。
固定测试不得导入实现的内部判定函数计算expected。变异实验应修改单一错误并确认对应断言失败。
