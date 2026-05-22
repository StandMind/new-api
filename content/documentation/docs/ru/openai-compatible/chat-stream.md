Streaming-запросы возвращают Server-Sent Events с постепенным выводом. Используйте их для chat UI, CLI и длинных ответов.

**Endpoint:** `POST /v1/chat/completions`

- Читайте строки data и добавляйте delta.content.
- data: [DONE] означает конец.
- stream_options.include_usage может вернуть usage в конце.
- Не повторяйте прерванный запрос бесконечно.

```
curl https://aivrae.com/v1/chat/completions \
  -H "Authorization: Bearer YOUR_AIVRAE_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "MODEL_NAME",
    "stream": true,
    "stream_options": { "include_usage": true },
    "messages": [
      { "role": "user", "content": "Предложи три идеи названия продукта." }
    ]
  }'
```
