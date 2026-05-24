流式 Chat Completions 使用 Server-Sent Events 持续返回增量内容。它适合聊天界面、命令行实时输出、长文本生成和需要尽快看到首字响应的场景。

Endpoint: `POST /v1/chat/completions`

> [!TIP]
> 文档页底部的 API 调试器现在支持读取流式响应。请求体中设置 `"stream": true` 后，响应面板会随着 SSE 分片实时追加内容。

## 请求体参数

| 参数 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| model | string | 是 | 要调用的模型名称，例如 `gpt-5.4-mini`。 |
| messages | array | 是 | 对话消息数组，结构与非流式接口一致。 |
| stream | boolean | 是 | 必须为 `true`，表示使用流式输出。 |
| stream_options | object | 否 | 流式附加选项。 |
| stream_options.include_usage | boolean | 否 | 为 `true` 时，上游支持的情况下会在结束前返回用量统计。 |
| temperature | number | 否 | 控制输出随机性。 |
| top_p | number | 否 | 核采样参数。 |
| max_tokens | integer | 否 | 本次响应最多生成的 token 数。示例使用 `4096`。 |
| tools | array | 否 | 工具定义；流式工具调用会通过 `delta.tool_calls` 增量返回。 |
| tool_choice | string/object | 否 | 控制工具调用策略。 |
| response_format | object | 否 | 指定输出格式。使用 JSON 格式时仍需在提示词里明确要求返回 JSON。 |

## SSE 返回格式

| 字段 | 说明 |
| --- | --- |
| data | 每个 SSE 事件的数据行。通常是一段 JSON，也可能是 `[DONE]`。 |
| choices[].delta.content | 本次分片新增的文本内容，需要追加到当前回答。 |
| choices[].delta.role | 流式开始时可能返回角色信息。 |
| choices[].delta.tool_calls | 工具调用的增量片段，需要按 `index` 合并。 |
| choices[].finish_reason | 结束原因。非空时表示该 choice 已结束。 |
| usage | 当 `stream_options.include_usage=true` 且上游支持时，可能在末尾返回。 |

## 示例请求

```bash
curl https://aivrae.com/v1/chat/completions \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-5.4-mini",
    "stream": true,
    "stream_options": { "include_usage": true },
    "messages": [
      { "role": "user", "content": "Give me three product name ideas." }
    ],
    "max_tokens": 4096
  }'
```

## 示例响应片段

```text
data: {"choices":[{"delta":{"role":"assistant"},"index":0}]}

data: {"choices":[{"delta":{"content":"第一"},"index":0}]}

data: {"choices":[{"delta":{"content":"个想法"},"index":0}]}

data: [DONE]
```

## 调用建议

- 客户端逐行读取 `data:`，把 `delta.content` 追加到当前回答。
- 遇到 `data: [DONE]` 后关闭读取。
- 网络中断时不要无条件重试同一个生成请求，避免重复生成和重复计费。
- 如果响应提前结束且 `finish_reason` 为 `length`，请调高 `max_tokens` 或缩短输入。

## 官方文档

- [OpenAI Chat Completions API](https://platform.openai.com/docs/api-reference/chat/create)
- [OpenAI Streaming guide](https://platform.openai.com/docs/guides/streaming-responses)
