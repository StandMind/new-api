Aivrae 默认推荐使用 OpenAI 兼容的 Chat Completions 格式。它适合绝大多数文本聊天模型、SDK、客户端和自动化工具。部分模型族也支持原生格式，例如 Claude Messages 和 Gemini generateContent。

| 调用格式 | 适用模型 | 接口 | 认证方式 | 建议用途 |
| --- | --- | --- | --- | --- |
| OpenAI 兼容格式 | 控制台中可见的大多数文本聊天模型 | `/v1/chat/completions` | `Authorization: Bearer ...` | 默认优先选择，客户端兼容性最好。 |
| Claude 原生格式 | Claude 系列模型 | `/v1/messages` | `x-api-key: ...` | Claude Code、Claude 原生 SDK 或需要 Anthropic Messages 参数的场景。 |
| Gemini 原生格式 | Gemini 系列模型 | `/v1beta/models/{model}:generateContent` | `Authorization: Bearer ...` | Gemini CLI、Gemini 原生 SDK 或需要 `contents/parts` 请求结构的场景。 |

## 兼容性原则

- 原生格式通常只适用于对应模型族：Claude 格式用于 Claude 模型，Gemini 格式用于 Gemini 模型。
- OpenAI 兼容格式是默认推荐入口，但不应假设每一个模型都支持所有 OpenAI 参数。
- Embedding 模型应使用 `/v1/embeddings`，不要用聊天接口调用。
- 是否可用以控制台模型权限和实际接口响应为准。

## 选择建议

- 不确定用什么格式时，先使用 `/v1/chat/completions`。
- 使用通用客户端时，优先选择 OpenAI Compatible Provider。
- 使用 Claude Code 或 Claude 原生 SDK 时，选择 `/v1/messages`。
- 使用 Gemini 原生工具链时，选择 `/v1beta/models/{model}:generateContent`。
