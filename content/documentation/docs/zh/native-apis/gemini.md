如果账号开放 Gemini 原生格式，可以使用 Google Gemini generateContent 风格请求。Gemini 原生格式通常只适用于 Gemini 系列模型；如果只需要通用客户端兼容性，仍建议优先使用 OpenAI 兼容 Chat Completions。

**Endpoint:** `POST /v1beta/models/{model}:generateContent`

| 请求头/参数 | 说明 |
| --- | --- |
| Authorization | 使用 `Bearer YOUR_AIVRAE_API_KEY`。 |
| model | 放在 URL 路径中，例如 `gemini-2.5-flash`，必须以控制台为准。 |
| contents | Gemini 原生输入数组，由 role 和 parts 组成。 |
| generationConfig | 温度、最大输出等生成参数。 |

```
curl https://aivrae.com/v1beta/models/GEMINI_MODEL_NAME:generateContent \
  -H "Authorization: Bearer YOUR_AIVRAE_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "contents": [
      {
        "role": "user",
        "parts": [
          { "text": "Write a concise project summary." }
        ]
      }
    ],
    "generationConfig": {
      "temperature": 0.7,
      "maxOutputTokens": 512
    }
  }'
```
