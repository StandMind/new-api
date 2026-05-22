Claude ネイティブアクセスが有効な場合、Anthropic Messages プロトコルを使えます。この形式は通常 Claude モデル専用です。一般的な互換性には Chat Completions を推奨します。

**Endpoint:** `POST /v1/messages`

| ヘッダー / パラメータ | 説明 |
| --- | --- |
| x-api-key | Aivrae API キー。 |
| anthropic-version | 例：2023-06-01。 |
| model | コンソールの Claude モデル名。 |
| max_tokens | 通常必要。 |
| messages | Anthropic messages 形式。 |

```
curl https://aivrae.com/v1/messages \
  -H "x-api-key: YOUR_AIVRAE_API_KEY" \
  -H "anthropic-version: 2023-06-01" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "CLAUDE_MODEL_NAME",
    "max_tokens": 512,
    "messages": [
      { "role": "user", "content": "プロジェクトの概要を簡潔に書いてください。" }
    ]
  }'
```
