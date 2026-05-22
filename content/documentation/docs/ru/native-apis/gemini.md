Если Gemini native доступен, используйте запросы в стиле Google Gemini generateContent. Этот формат обычно применим только к Gemini-моделям. Для широкой совместимости рекомендуется Chat Completions.

**Endpoint:** `POST /v1beta/models/{model}:generateContent`

| Заголовок / параметр | Описание |
| --- | --- |
| Authorization | `Bearer YOUR_AIVRAE_API_KEY`. |
| model | В пути URL, например `gemini-2.5-flash`. |
| contents | Входной массив Gemini с role и parts. |
| generationConfig | Параметры генерации. |

```
curl https://aivrae.com/v1beta/models/GEMINI_MODEL_NAME:generateContent \
  -H "Authorization: Bearer YOUR_AIVRAE_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "contents": [
      {
        "role": "user",
        "parts": [
          { "text": "Напиши краткое описание проекта." }
        ]
      }
    ],
    "generationConfig": {
      "temperature": 0.7,
      "maxOutputTokens": 512
    }
  }'
```
