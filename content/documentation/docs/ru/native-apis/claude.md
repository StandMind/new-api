Если Claude native доступен, используйте протокол Anthropic Messages. Этот формат обычно применим только к Claude-моделям. Для широкой совместимости рекомендуется Chat Completions.

**Endpoint:** `POST /v1/messages`

| Заголовок / параметр | Описание |
| --- | --- |
| x-api-key | Ваш Aivrae API key. |
| anthropic-version | Версия клиента, например 2023-06-01. |
| model | Claude-модель из консоли. |
| max_tokens | Обычно обязателен. |
| messages | Формат Anthropic messages. |

```
curl https://aivrae.com/v1/messages \
  -H "x-api-key: YOUR_AIVRAE_API_KEY" \
  -H "anthropic-version: 2023-06-01" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "CLAUDE_MODEL_NAME",
    "max_tokens": 512,
    "messages": [
      { "role": "user", "content": "Напиши краткое описание проекта." }
    ]
  }'
```
