Completions は古いテキスト補完形式です。古いクライアントや特定モデルが必要としない限り、Chat Completions を優先してください。

**Endpoint:** `POST /v1/completions`

```
curl https://aivrae.com/v1/completions \
  -H "Authorization: Bearer YOUR_AIVRAE_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "MODEL_NAME",
    "prompt": "API ゲートウェイ向けの短いキャッチコピーを書いてください。",
    "max_tokens": 4096
  }'
```
