Aivrae 文本接口使用 HTTPS JSON 请求。请求需要包含认证信息、模型名称和输入内容；响应结构取决于你选择的兼容协议。

| 项目 | 填写方式 | 备注 |
| --- | --- | --- |
| 认证 | Authorization: Bearer YOUR_AIVRAE_API_KEY | OpenAI 兼容接口和 Gemini 原生格式使用 Bearer 认证。 |
| Claude 认证 | x-api-key: YOUR_AIVRAE_API_KEY | Claude 原生格式使用 Anthropic 风格请求头。 |
| 内容类型 | Content-Type: application/json | 文本接口均使用 JSON。 |
| 模型名称 | 从控制台复制 | 不要手写猜测模型名，大小写和后缀必须一致。 |
| 超时 | 建议 60 秒以上 | 流式请求需要客户端持续读取，不要过早断开。 |
| 日志 | 记录时间、模型、状态码、错误消息 | 排查扣费、限流和服务异常时非常关键。 |
