Текстовые API Aivrae используют HTTPS JSON. Укажите аутентификацию, модель и входные данные.

| Пункт | Значение | Примечания |
| --- | --- | --- |
| Аутентификация | Authorization: Bearer YOUR_AIVRAE_API_KEY | Для OpenAI-совместимых endpoint и Gemini native. |
| Claude auth | x-api-key: YOUR_AIVRAE_API_KEY | Для Claude native. |
| Content type | Content-Type: application/json | Текстовые endpoint используют JSON. |
| Имя модели | Из консоли | Регистр и суффиксы важны. |
| Timeout | 60 секунд или больше | Streaming держит соединение открытым. |
| Логи | Время, модель, статус, ошибка | Важно для биллинга и диагностики. |
