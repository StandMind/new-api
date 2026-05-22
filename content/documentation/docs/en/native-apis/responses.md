Responses is a newer unified input/output format. Some newer models may prefer Responses. If your client does not support it, continue using Chat Completions.

**Endpoint:** `POST /v1/responses`

| Item | Chat Completions | Responses |
| --- | --- | --- |
| Input field | messages | input |
| Typical use | Traditional chat clients and plugins | Newer models, tools, and unified response workflows |
| Streaming | stream: true | stream when supported |
| Recommendation | Default choice | Use when model or tool requires it |

```
curl https://aivrae.com/v1/responses \
  -H "Authorization: Bearer YOUR_AIVRAE_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "MODEL_NAME",
    "input": [
      { "role": "user", "content": "Summarize the benefits of API gateways." }
    ]
  }'
```
