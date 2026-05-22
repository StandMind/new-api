Function calling и структурированный вывод относятся к текстовым API, но зависят от модели и маршрута. Добавьте fallback для неподдерживаемых моделей.

**Endpoint:** `POST /v1/chat/completions`

## Пример function calling

```
curl https://aivrae.com/v1/chat/completions \
  -H "Authorization: Bearer YOUR_AIVRAE_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "MODEL_NAME",
    "messages": [
      { "role": "user", "content": "Какая сегодня погода в Москве?" }
    ],
    "tools": [
      {
        "type": "function",
        "function": {
          "name": "get_weather",
          "description": "Получить текущую погоду",
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

## Пример JSON

```
curl https://aivrae.com/v1/chat/completions \
  -H "Authorization: Bearer YOUR_AIVRAE_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "MODEL_NAME",
    "response_format": { "type": "json_object" },
    "messages": [
      { "role": "system", "content": "Возвращай только корректный JSON." },
      { "role": "user", "content": "Создай JSON-объект с name и slogan." }
    ]
  }'
```
