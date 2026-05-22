ストリーミングは Server-Sent Events で増分出力します。チャット UI、CLI、長い回答に適しています。

**Endpoint:** `POST /v1/chat/completions`

- data 行を読み、delta.content を追加します。
- data: [DONE] が終了を示します。
- include_usage は対応時に最後に usage を返します。
- 中断した同じリクエストを無制限に再試行しないでください。

```
curl https://aivrae.com/v1/chat/completions \
  -H "Authorization: Bearer YOUR_AIVRAE_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "MODEL_NAME",
    "stream": true,
    "stream_options": { "include_usage": true },
    "messages": [
      { "role": "user", "content": "プロダクト名の案を3つ出してください。" }
    ]
  }'
```
