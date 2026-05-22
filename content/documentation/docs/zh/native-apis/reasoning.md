推理模型可能返回 reasoning_content、reasoning、thinking 或 <think>...</think> 包裹的内容。不同模型和路由字段不完全一致。

| 字段 / 形式 | 可能来源 | 处理建议 |
| --- | --- | --- |
| delta.reasoning_content | DeepSeek 等推理模型 | 可单独展示或折叠。 |
| reasoning / thinking | Gemini 或部分兼容路由 | 不要假设所有模型都有该字段。 |
| <think>...</think> | 部分兼容或反代模型 | 如果不展示推理，应在 UI 层过滤。 |
| delta.content | 最终回答内容 | 面向用户展示的主文本。 |
