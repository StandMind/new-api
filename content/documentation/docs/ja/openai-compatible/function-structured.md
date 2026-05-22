関数呼び出しと構造化出力はテキスト API ですが、モデル能力とルート互換性に依存します。非対応モデル用の fallback を用意してください。

**Endpoint:** `POST /v1/chat/completions`

## 関数呼び出し例

```
curl https://aivrae.com/v1/chat/completions \
  -H "Authorization: Bearer YOUR_AIVRAE_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "MODEL_NAME",
    "messages": [
      { "role": "user", "content": "東京の今日の天気は？" }
    ],
    "tools": [
      {
        "type": "function",
        "function": {
          "name": "get_weather",
          "description": "現在の天気を取得",
          "parameters": {
            "type": "object",
            "properties": {
              "city": { "type": "string" }
            },
            "required": ["city"]
          }
        }
      }
    ],
    "tool_choice": "auto"
  }'
```

## JSON 例

```
curl https://aivrae.com/v1/chat/completions \
  -H "Authorization: Bearer YOUR_AIVRAE_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "MODEL_NAME",
    "response_format": { "type": "json_object" },
    "messages": [
      { "role": "system", "content": "有効な JSON だけを返してください。" },
      { "role": "user", "content": "name と slogan を含む JSON オブジェクトを生成してください。" }
    ]
  }'
```
