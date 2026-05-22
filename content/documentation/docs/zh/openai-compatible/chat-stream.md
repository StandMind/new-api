流式请求会返回 Server-Sent Events，并逐步输出内容。适合聊天 UI、命令行实时输出和较长回答。

**Endpoint:** `POST /v1/chat/completions`

- 读取 data 行，并把 delta.content 追加到当前回答。
- data: [DONE] 表示响应结束。
- stream_options.include_usage 在支持时可能会在末尾返回用量。
- 网络中断时不要无限重试同一请求，避免重复生成和重复扣费。

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
