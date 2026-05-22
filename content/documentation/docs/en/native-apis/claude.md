If Claude native access is enabled, use the Anthropic Messages protocol. Claude native format usually applies only to Claude models. For broad client compatibility, OpenAI-compatible Chat Completions remain the recommended default.

**Endpoint:** `POST /v1/messages`

| Header / parameter | Description |
| --- | --- |
| x-api-key | Your Aivrae API key. |
| anthropic-version | Use your client's default version, such as 2023-06-01. |
| model | Claude model name copied from the console. |
| max_tokens | Usually required for Claude native requests. |
| messages | Anthropic messages format. |

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
