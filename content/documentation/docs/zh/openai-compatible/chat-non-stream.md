非流式 Chat Completions 会在模型生成结束后一次性返回完整 JSON。它适合短文本生成、分类、摘要、结构化抽取，以及后台任务中不需要实时展示生成过程的场景。

Endpoint: `POST /v1/chat/completions`

> [!NOTE]
> 请求格式遵循 OpenAI Chat Completions 风格。模型名、可用参数和计费以控制台中实际开放的模型与上游渠道为准。

## 请求头

| 参数 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| Authorization | string | 是 | 使用 `Bearer YOUR_API_KEY`。 |
| Content-Type | string | 是 | 固定为 `application/json`。 |

## 请求体参数

| 参数 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| model | string | 是 | 要调用的模型名称，例如 `gpt-5.4-mini`。请使用模型广场或控制台中可用的模型名。 |
| messages | array | 是 | 对话消息数组，会按顺序组成上下文。 |
| messages[].role | string | 是 | 消息角色，常用值为 `system`、`user`、`assistant`、`tool`。 |
| messages[].content | string/array | 是 | 消息内容。纯文本可直接传字符串；多模态模型可传内容数组。 |
| temperature | number | 否 | 采样随机性，通常为 `0` 到 `2`。值越高越发散，越低越稳定。 |
| top_p | number | 否 | 核采样参数。通常不要和 `temperature` 同时大幅调整。 |
| max_tokens | integer | 否 | 本次响应最多生成的 token 数。示例使用 `4096`，实际上限受模型上下文和上游限制影响。 |
| stream | boolean | 否 | 非流式请求不传或传 `false`。需要实时输出时使用流式接口。 |
| stop | string/array | 否 | 遇到指定文本时停止生成。 |
| tools | array | 否 | 函数调用或工具定义，支持情况取决于模型。 |
| tool_choice | string/object | 否 | 控制工具调用策略，例如 `auto`、`none` 或指定某个工具。 |
| response_format | object | 否 | 要求模型返回指定格式，例如 JSON 对象或 JSON Schema。 |
| presence_penalty | number | 否 | 降低重复话题的概率，取值通常为 `-2` 到 `2`。 |
| frequency_penalty | number | 否 | 降低重复词句的概率，取值通常为 `-2` 到 `2`。 |
| user | string | 否 | 终端用户标识，可用于审计和风控。 |

## 示例请求

```bash
curl https://aivrae.com/v1/chat/completions \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-5.4-mini",
    "messages": [
      { "role": "system", "content": "You are a concise assistant." },
      { "role": "user", "content": "Write a short welcome message." }
    ],
    "temperature": 0.7,
    "max_tokens": 4096
  }'
```

## 响应字段

| 字段 | 说明 |
| --- | --- |
| id | 本次请求的响应 ID。 |
| object | 对象类型，通常为 `chat.completion`。 |
| created | 响应创建时间戳。 |
| model | 实际返回的模型名。 |
| choices[].message.role | 返回消息角色，通常为 `assistant`。 |
| choices[].message.content | 模型生成的主要文本。 |
| choices[].finish_reason | 结束原因，常见值包括 `stop`、`length`、`tool_calls`。如果是 `length`，说明命中了输出长度限制。 |
| usage.prompt_tokens | 输入 token 数。 |
| usage.completion_tokens | 输出 token 数。 |
| usage.total_tokens | 输入和输出 token 总数，可用于排查计费。 |

## 官方文档

- [OpenAI Chat Completions API](https://platform.openai.com/docs/api-reference/chat/create)
- [OpenAI Text generation guide](https://platform.openai.com/docs/guides/text-generation)
