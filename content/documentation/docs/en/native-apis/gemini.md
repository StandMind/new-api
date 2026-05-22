If Gemini native access is enabled, use Google Gemini generateContent-style requests. Gemini native format usually applies only to Gemini models. For broad client compatibility, OpenAI-compatible Chat Completions remain the recommended default.

**Endpoint:** `POST /v1beta/models/{model}:generateContent`

| Header / parameter | Description |
| --- | --- |
| Authorization | Use `Bearer YOUR_AIVRAE_API_KEY`. |
| model | Placed in the URL path, for example `gemini-2.5-flash`; copy it from the console. |
| contents | Gemini native input array with role and parts. |
| generationConfig | Generation settings such as temperature and max output tokens. |

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
