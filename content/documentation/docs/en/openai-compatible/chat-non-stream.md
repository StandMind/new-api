Non-streaming Chat Completions return one complete JSON response after generation finishes. Use this mode for short text generation, classification, summaries, structured extraction, and background jobs that do not need token-by-token display.

Endpoint: `POST /v1/chat/completions`

> [!NOTE]
> The request shape follows the OpenAI Chat Completions style. Available models, supported parameters, and billing depend on the models and upstream channels enabled in your console.

## Headers

| Parameter | Type | Required | Description |
| --- | --- | --- | --- |
| Authorization | string | Yes | Use `Bearer YOUR_API_KEY`. |
| Content-Type | string | Yes | Must be `application/json`. |

## Request Body

| Parameter | Type | Required | Description |
| --- | --- | --- | --- |
| model | string | Yes | Model name to call, for example `gpt-5.4-mini`. Use a model that is available in the model list or console. |
| messages | array | Yes | Ordered conversation messages used as context. |
| messages[].role | string | Yes | Message role. Common values are `system`, `user`, `assistant`, and `tool`. |
| messages[].content | string/array | Yes | Message content. Use a string for plain text or an array for multimodal content on supported models. |
| temperature | number | No | Sampling randomness, usually from `0` to `2`. Higher values are more creative; lower values are more deterministic. |
| top_p | number | No | Nucleus sampling value. Avoid changing both `temperature` and `top_p` aggressively at the same time. |
| max_tokens | integer | No | Maximum tokens to generate. The example uses `4096`; the real limit depends on model context and upstream limits. |
| stream | boolean | No | Omit it or set `false` for non-streaming requests. Use the streaming endpoint behavior for realtime output. |
| stop | string/array | No | Stop generation when any specified sequence appears. |
| tools | array | No | Function or tool definitions, depending on model support. |
| tool_choice | string/object | No | Controls tool selection, such as `auto`, `none`, or a specific tool. |
| response_format | object | No | Requests a specific output format, such as JSON object or JSON Schema. |
| presence_penalty | number | No | Penalizes repeated topics. Values are usually between `-2` and `2`. |
| frequency_penalty | number | No | Penalizes repeated wording. Values are usually between `-2` and `2`. |
| user | string | No | End-user identifier for audit and risk controls. |

## Example Request

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

## Response Fields

| Field | Description |
| --- | --- |
| id | Response identifier. |
| object | Object type, usually `chat.completion`. |
| created | Unix timestamp when the response was created. |
| model | Model name that produced the response. |
| choices[].message.role | Returned message role, usually `assistant`. |
| choices[].message.content | Main generated text. |
| choices[].finish_reason | Stop reason such as `stop`, `length`, or `tool_calls`. `length` means the output limit was reached. |
| usage.prompt_tokens | Input token count. |
| usage.completion_tokens | Output token count. |
| usage.total_tokens | Total input and output tokens, useful for billing checks. |

## Official Docs

- [OpenAI Chat Completions API](https://platform.openai.com/docs/api-reference/chat/create)
- [OpenAI Text generation guide](https://platform.openai.com/docs/guides/text-generation)
