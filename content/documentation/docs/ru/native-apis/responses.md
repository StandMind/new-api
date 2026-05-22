Responses — более новый единый формат ввода/вывода. Некоторые модели и инструменты могут предпочитать его. Если клиент не поддерживает его, используйте Chat Completions.

**Endpoint:** `POST /v1/responses`

| Пункт | Chat Completions | Responses |
| --- | --- | --- |
| Поле ввода | messages | input |
| Использование | Обычные chat-клиенты | Новые модели и единые workflow |
| Streaming | stream: true | stream при поддержке |
| Рекомендация | По умолчанию | Когда требуется моделью или инструментом |

```
curl https://aivrae.com/v1/responses \
  -H "Authorization: Bearer YOUR_AIVRAE_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "MODEL_NAME",
    "input": [
      { "role": "user", "content": "Summarize the benefits of API gateways." }
    ]
  }'
```
