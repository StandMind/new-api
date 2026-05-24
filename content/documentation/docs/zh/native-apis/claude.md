Claude 原生接口使用 Anthropic Messages 协议，适合已经接入 Claude SDK 或需要使用 Claude 原生 `system`、`tools`、`thinking`、`stream` 等能力的场景。只需要通用兼容性时，建议优先使用 OpenAI 兼容的 Chat Completions。

Endpoint: `POST /v1/messages`

> [!NOTE]
> Claude 原生接口使用 `x-api-key` 请求头，而不是 `Authorization: Bearer ...`。调试器会按该接口自动设置鉴权头。

## 请求头

| 参数 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| x-api-key | string | 是 | 填写你的 API Key。 |
| anthropic-version | string | 是 | Anthropic API 版本，示例使用 `2023-06-01`。 |
| Content-Type | string | 是 | 固定为 `application/json`。 |

## 请求体参数

| 参数 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| model | string | 是 | Claude 模型名称。请使用控制台中可用的模型名。 |
| max_tokens | integer | 是 | 最大输出 token 数。示例使用 `4096`。 |
| messages | array | 是 | 对话消息数组。 |
| messages[].role | string | 是 | 消息角色，通常为 `user` 或 `assistant`。 |
| messages[].content | string/array | 是 | 消息内容。文本可直接传字符串；多模态可传内容块数组。 |
| system | string/array | 否 | 系统提示词，用于设定助手行为。 |
| temperature | number | 否 | 控制随机性。 |
| top_p | number | 否 | 核采样参数。 |
| top_k | integer | 否 | 限制候选 token 数。 |
| stop_sequences | array | 否 | 遇到指定文本时停止生成。 |
| stream | boolean | 否 | 为 `true` 时使用 Claude 原生流式返回。 |
| tools | array | 否 | 工具定义。 |
| tool_choice | object | 否 | 控制工具选择策略。 |
| metadata | object | 否 | 可选的用户标识或业务元信息。 |
| thinking | object | 否 | 扩展思考能力配置，只有支持的模型会生效。 |

## 示例请求

```bash
curl https://aivrae.com/v1/messages \
  -H "x-api-key: YOUR_API_KEY" \
  -H "anthropic-version: 2023-06-01" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "claude-3-5-sonnet-20241022",
    "max_tokens": 4096,
    "messages": [
      { "role": "user", "content": "Write a concise project summary." }
    ]
  }'
```

## 响应字段

| 字段 | 说明 |
| --- | --- |
| id | Claude 响应 ID。 |
| type | 对象类型，通常为 `message`。 |
| role | 返回消息角色，通常为 `assistant`。 |
| content | 内容块数组，文本通常在 `content[].text`。 |
| model | 实际返回的模型名。 |
| stop_reason | 停止原因，例如 `end_turn`、`max_tokens`、`tool_use`。 |
| usage.input_tokens | 输入 token 数。 |
| usage.output_tokens | 输出 token 数。 |

## 官方文档

- [Anthropic Messages API](https://docs.anthropic.com/en/api/messages)
- [Anthropic streaming Messages](https://docs.anthropic.com/en/api/messages-streaming)
