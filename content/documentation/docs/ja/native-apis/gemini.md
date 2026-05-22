Gemini ネイティブアクセスが有効な場合、Google Gemini generateContent 形式のリクエストを使えます。この形式は通常 Gemini モデル専用です。一般的な互換性には Chat Completions を推奨します。

**Endpoint:** `POST /v1beta/models/{model}:generateContent`

| ヘッダー / パラメータ | 説明 |
| --- | --- |
| Authorization | `Bearer YOUR_AIVRAE_API_KEY`。 |
| model | URL パス内のモデル名。例：`gemini-2.5-flash`。 |
| contents | role と parts を含む Gemini 入力配列。 |
| generationConfig | temperature や maxOutputTokens。 |

```
curl https://aivrae.com/v1beta/models/GEMINI_MODEL_NAME:generateContent \
  -H "Authorization: Bearer YOUR_AIVRAE_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "contents": [
      {
        "role": "user",
        "parts": [
          { "text": "プロジェクトの概要を簡潔に書いてください。" }
        ]
      }
    ],
    "generationConfig": {
      "temperature": 0.7,
      "maxOutputTokens": 512
    }
  }'
```
