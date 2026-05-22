Нестриминые запросы возвращают полный результат после генерации. Они подходят для коротких текстов, фоновых задач, классификации и резюме.

**Endpoint:** `POST /v1/chat/completions`

| Параметр | Обязателен | Описание |
| --- | --- | --- |
| model | Да | Имя модели из консоли. |
| messages | Да | Массив сообщений с ролями system, user, assistant. |
| temperature | Нет | От 0 до 2; обычно 0.2-0.8. |
| max_tokens | Нет | Ограничивает длину вывода. |
| stream | Нет | Для нестриминга не указывайте или false. |

```
curl https://aivrae.com/v1/chat/completions \
  -H "Authorization: Bearer YOUR_AIVRAE_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "MODEL_NAME",
    "messages": [
      { "role": "system", "content": "Ты отвечаешь кратко." },
      { "role": "user", "content": "Напиши короткое приветственное сообщение." }
    ],
    "temperature": 0.7,
    "max_tokens": 512
  }'
```

## Частые поля ответа

| Поле | Описание |
| --- | --- |
| choices[0].message.content | Основной текст ответа модели. |
| choices[0].finish_reason | stop, length, tool_calls или похожая причина завершения. |
| usage.prompt_tokens | Количество входных токенов. |
| usage.completion_tokens | Количество выходных токенов. |
| usage.total_tokens | Общее количество токенов, часто используется для биллинга. |
