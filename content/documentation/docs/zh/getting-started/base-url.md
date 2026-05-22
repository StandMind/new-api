大多数客户端只需要填写 Base URL；少数客户端要求完整接口地址。优先使用 /v1 形式，这样 SDK 会自动拼接具体路径。

| 使用场景 | 推荐填写 | 说明 |
| --- | --- | --- |
| OpenAI SDK / 大多数客户端 | https://aivrae.com/v1 | 最推荐，兼容性最好。 |
| 客户端要求完整聊天接口 | https://aivrae.com/v1/chat/completions | 仅在客户端明确要求时使用。 |
| 模型列表 | https://aivrae.com/v1/models | 用于检查 Key 可见模型。 |
| Claude 原生格式 | https://aivrae.com/v1/messages | 需要使用 x-api-key 和 anthropic-version 请求头。 |
| Gemini 原生格式 | https://aivrae.com/v1beta/models/{model}:generateContent | 将 {model} 替换为控制台可见的 Gemini 模型名。 |

```
Recommended Base URL
https://aivrae.com/v1

Common full endpoints
POST https://aivrae.com/v1/chat/completions
POST https://aivrae.com/v1/responses
POST https://aivrae.com/v1/completions
POST https://aivrae.com/v1/embeddings
GET  https://aivrae.com/v1/models

Native text endpoints
POST https://aivrae.com/v1/messages
POST https://aivrae.com/v1beta/models/{model}:generateContent
```
