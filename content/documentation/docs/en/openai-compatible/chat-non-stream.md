Non-streaming requests return the full result after generation. Use them for short text, background jobs, classification, summaries, and workflows that do not need live output.

**Endpoint:** `POST /v1/chat/completions`

| Parameter | Required | Description |
| --- | --- | --- |
| model | Yes | Model name copied from the console. |
| messages | Yes | Conversation array with roles such as system, user, assistant. |
| temperature | No | 0 to 2; common values are 0.2 to 0.8. |
| max_tokens | No | Limits output length within model context limits. |
| stream | No | Omit or set false for non-streaming. |

```
curl https://aivrae.com/v1/chat/completions \
  -H "Authorization: Bearer YOUR_AIVRAE_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "MODEL_NAME",
    "messages": [
      { "role": "system", "content": "You are a concise assistant." },
      { "role": "user", "content": "Write a short welcome message." }
    ],
    "temperature": 0.7,
    "max_tokens": 512
  }'
```

## Common response fields

| Field | Description |
| --- | --- |
| choices[0].message.content | Main model output. |
| choices[0].finish_reason | stop, length, tool_calls, or similar reason. |
| usage.prompt_tokens | Input token count. |
| usage.completion_tokens | Output token count. |
| usage.total_tokens | Total tokens, often used as billing reference. |
