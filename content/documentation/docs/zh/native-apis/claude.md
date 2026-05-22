如果账号开放 Claude 原生格式，可以使用 Anthropic Messages 协议。Claude 原生格式通常只适用于 Claude 系列模型；如果只需要通用客户端兼容性，仍建议优先使用 OpenAI 兼容 Chat Completions。

**Endpoint:** `POST /v1/messages`

| 请求头/参数 | 说明 |
| --- | --- |
| x-api-key | 填写你的 Aivrae API Key。 |
| anthropic-version | 建议使用客户端默认版本，例如 2023-06-01。 |
| model | Claude 模型名称，必须以控制台为准。 |
| max_tokens | Claude 原生请求通常需要显式设置。 |
| messages | Anthropic messages 格式。 |

```
curl https://aivrae.com/v1/messages \
  -H "x-api-key: YOUR_AIVRAE_API_KEY" \
  -H "anthropic-version: 2023-06-01" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "CLAUDE_MODEL_NAME",
    "max_tokens": 512,
    "messages": [
      { "role": "user", "content": "Write a concise project summary." }
    ]
  }'
```
