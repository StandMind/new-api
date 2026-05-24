Streaming Chat Completions return incremental Server-Sent Events while the model is generating. Use this mode for chat UIs, realtime terminal output, long answers, and cases where first-token latency matters.

Endpoint: `POST /v1/chat/completions`

> [!TIP]
> The API tester at the bottom of this page now reads streaming responses. Set `"stream": true` in the request body and chunks will appear in the response panel as they arrive.

## Request Body

| Parameter | Type | Required | Description |
| --- | --- | --- | --- |
| model | string | Yes | Model name to call, for example `gpt-5.4-mini`. |
| messages | array | Yes | Ordered conversation messages. Same structure as non-streaming Chat Completions. |
| stream | boolean | Yes | Must be `true` to enable streaming output. |
| stream_options | object | No | Additional streaming options. |
| stream_options.include_usage | boolean | No | When `true`, supported upstreams may send usage data near the end of the stream. |
| temperature | number | No | Controls randomness. |
| top_p | number | No | Nucleus sampling value. |
| max_tokens | integer | No | Maximum tokens to generate. The example uses `4096`. |
| tools | array | No | Tool definitions. Streaming tool calls are returned through `delta.tool_calls`. |
| tool_choice | string/object | No | Controls tool selection. |
| response_format | object | No | Requests a specific output format. For JSON output, also tell the model to return JSON in the prompt. |

## SSE Response Shape

| Field | Description |
| --- | --- |
| data | SSE data line. It is usually JSON and may also be `[DONE]`. |
| choices[].delta.content | Newly generated text for this chunk. Append it to the current answer. |
| choices[].delta.role | Role information that may appear at the beginning of the stream. |
| choices[].delta.tool_calls | Incremental tool-call fragments. Merge them by `index`. |
| choices[].finish_reason | Stop reason. A non-empty value means that choice has finished. |
| usage | May appear near the end when `stream_options.include_usage=true` and the upstream supports it. |

## Example Request

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

## Example Stream

```text
data: {"choices":[{"delta":{"role":"assistant"},"index":0}]}

data: {"choices":[{"delta":{"content":"First"},"index":0}]}

data: {"choices":[{"delta":{"content":" idea"},"index":0}]}

data: [DONE]
```

## Client Notes

- Read each `data:` line and append `delta.content` to the current answer.
- Close the reader when you receive `data: [DONE]`.
- Do not retry the same generation request indefinitely after a network interruption, or you may duplicate generation and billing.
- If the stream ends with `finish_reason: length`, increase `max_tokens` or shorten the prompt.

## Official Docs

- [OpenAI Chat Completions API](https://platform.openai.com/docs/api-reference/chat/create)
- [OpenAI Streaming guide](https://platform.openai.com/docs/guides/streaming-responses)
