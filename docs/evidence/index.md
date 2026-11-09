# 参考验证与个人证据入口

维护者参考运行见 [VALIDATION](../VALIDATION.md)，范围见 [COVERAGE](../COVERAGE.md)。原始文件生成在 artifacts/raw，默认不入 Git；CI 运行后上传 artifact。

个人证据由学习者追加，不把维护者参考运行复制成自己的掌握证明。每条至少包含：Lab/任务、Git SHA（dirty 必须标记）、命令、环境、退出码、原始文件 SHA256、预测、实际结果、解释、剩余限制、复现者。失败也保留。

建议阶段：第一次看演示 → Lab00 模型 → 每 Lab 一次反例 → 每两 Lab 改约束 → 三组 A/B 各三次 → 未知故障答辩 → 第二人复现。未执行统一写 not_run。
