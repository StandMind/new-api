The Claude native API uses the Anthropic Messages protocol. It is useful when your client already uses Claude SDKs or when you need native fields such as `system`, `tools`, `thinking`, and `stream`. For broad client compatibility, prefer the OpenAI-compatible Chat Completions API.

Endpoint: `POST /v1/messages`

> [!NOTE]
> Claude native requests use the `x-api-key` header instead of `Authorization: Bearer ...`. The tester sets the matching authentication header for this endpoint.

## Headers

| Parameter | Type | Required | Description |
| --- | --- | --- | --- |
| x-api-key | string | Yes | Your API key. |
| anthropic-version | string | Yes | Anthropic API version. The example uses `2023-06-01`. |
| Content-Type | string | Yes | Must be `application/json`. |

## Request Body

| Parameter | Type | Required | Description |
| --- | --- | --- | --- |
| model | string | Yes | Claude model name. Use a model that is enabled in your console. |
| max_tokens | integer | Yes | Maximum output tokens. The example uses `4096`. |
| messages | array | Yes | Conversation messages. |
| messages[].role | string | Yes | Message role, usually `user` or `assistant`. |
| messages[].content | string/array | Yes | Message content. Use a string for text or content blocks for multimodal requests. |
| system | string/array | No | System prompt that guides assistant behavior. |
| temperature | number | No | Controls randomness. |
| top_p | number | No | Nucleus sampling value. |
| top_k | integer | No | Limits candidate tokens. |
| stop_sequences | array | No | Stop generation when any sequence appears. |
| stream | boolean | No | Set `true` for Claude native streaming. |
| tools | array | No | Tool definitions. |
| tool_choice | object | No | Controls tool selection. |
| metadata | object | No | Optional user or business metadata. |
| thinking | object | No | Extended thinking configuration. Only supported models apply it. |

## Example Request

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

## Response Fields

| Field | Description |
| --- | --- |
| id | Claude response identifier. |
| type | Object type, usually `message`. |
| role | Returned role, usually `assistant`. |
| content | Content block array. Text is usually in `content[].text`. |
| model | Model that produced the response. |
| stop_reason | Stop reason, such as `end_turn`, `max_tokens`, or `tool_use`. |
| usage.input_tokens | Input token count. |
| usage.output_tokens | Output token count. |

## Official Docs

- [Anthropic Messages API](https://docs.anthropic.com/en/api/messages)
- [Anthropic streaming Messages](https://docs.anthropic.com/en/api/messages-streaming)
