Streaming requests return Server-Sent Events with incremental content. Use streaming for chat UIs, command-line output, and long answers.

**Endpoint:** `POST /v1/chat/completions`

- Read data lines and append delta.content to the current answer.
- data: [DONE] marks the end of the response.
- stream_options.include_usage may return usage at the end when supported.
- Do not retry interrupted requests forever; this can duplicate generation and cost.

```
curl https://aivrae.com/v1/chat/completions \
  -H "Authorization: Bearer YOUR_AIVRAE_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "MODEL_NAME",
    "stream": true,
    "stream_options": { "include_usage": true },
    "messages": [
      { "role": "user", "content": "Give me three product name ideas." }
    ]
  }'
```
