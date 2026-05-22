Responses は新しい統一入出力形式です。一部の新しいモデルやツールが好む場合があります。未対応クライアントでは Chat Completions を使ってください。

**Endpoint:** `POST /v1/responses`

| 項目 | Chat Completions | Responses |
| --- | --- | --- |
| 入力 | messages | input |
| 用途 | 従来の chat クライアント | 新モデルと統一 workflow |
| Streaming | stream: true | 対応時 stream |
| 推奨 | デフォルト | モデルやツールが要求する場合 |

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
