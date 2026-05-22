Responses 是较新的统一输入/输出格式。部分新模型或工具链可能更偏好 Responses。如果你的客户端暂不支持，继续使用 Chat Completions 即可。

**Endpoint:** `POST /v1/responses`

| 项目 | Chat Completions | Responses |
| --- | --- | --- |
| 输入字段 | messages | input |
| 常见用途 | 传统聊天客户端和插件 | 新模型、工具和统一响应工作流 |
| 流式 | stream: true | 支持时使用 stream |
| 建议 | 默认选择 | 模型或工具明确要求时使用 |

```
curl https://aivrae.com/v1/responses \
  -H "Authorization: Bearer YOUR_AIVRAE_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "MODEL_NAME",
    "input": [
      { "role": "user", "content": "Summarize the benefits of API gateways." }
    ]
  }'
```
