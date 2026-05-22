Большинству клиентов нужен только Base URL. Предпочитайте /v1, чтобы SDK сам добавлял пути.

| Сценарий | Значение | Примечания |
| --- | --- | --- |
| OpenAI SDK / клиенты | https://aivrae.com/v1 | Рекомендуется. |
| Полный chat endpoint | https://aivrae.com/v1/chat/completions | Только если клиент требует. |
| Список моделей | https://aivrae.com/v1/models | Проверка доступных моделей. |
| Claude native | https://aivrae.com/v1/messages | x-api-key и anthropic-version. |
| Gemini native | https://aivrae.com/v1beta/models/{model}:generateContent | Замените {model}. |

```
Рекомендуемый Base URL
https://aivrae.com/v1

Частые полные endpoints
POST https://aivrae.com/v1/chat/completions
POST https://aivrae.com/v1/responses
POST https://aivrae.com/v1/completions
POST https://aivrae.com/v1/embeddings
GET  https://aivrae.com/v1/models

Нативные текстовые endpoints
POST https://aivrae.com/v1/messages
POST https://aivrae.com/v1beta/models/{model}:generateContent
```
