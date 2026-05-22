Имена моделей должны точно совпадать с консолью. Доступные модели зависят от аккаунта, группы и плана; некоторые модели поддерживают только определенные endpoint или параметры.

**Endpoint:** `GET /v1/models`

| Поле / понятие | Описание |
| --- | --- |
| id | Значение для поля model. |
| owned_by / provider | Провайдер или тип маршрута, если указан. |
| Контекст | Смотрите консоль; слишком большой контекст может вернуть 400, 413 или ошибку сервиса. |
| Совместимость | Chat — chat completions, embeddings — embeddings, Claude — /v1/messages, Gemini — generateContent. |

```
curl https://aivrae.com/v1/models \
  -H "Authorization: Bearer YOUR_AIVRAE_API_KEY"
```
