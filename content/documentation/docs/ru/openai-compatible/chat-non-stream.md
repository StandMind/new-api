Chat Completions без потока возвращает полный JSON-ответ после завершения генерации. Этот режим подходит для коротких текстов, классификации, резюме, структурированного извлечения и фоновых задач без показа токенов в реальном времени.

Endpoint: `POST /v1/chat/completions`

> [!NOTE]
> Формат запроса совместим со стилем OpenAI Chat Completions. Доступные модели, параметры и тарификация зависят от моделей и upstream-каналов, включенных в консоли.

## Заголовки

| Параметр | Тип | Обязателен | Описание |
| --- | --- | --- | --- |
| Authorization | string | Да | Используйте `Bearer YOUR_API_KEY`. |
| Content-Type | string | Да | Должен быть `application/json`. |

## Тело запроса

| Параметр | Тип | Обязателен | Описание |
| --- | --- | --- | --- |
| model | string | Да | Имя модели, например `gpt-5.4-mini`. Используйте модель, доступную в списке моделей или консоли. |
| messages | array | Да | Упорядоченный массив сообщений диалога. |
| messages[].role | string | Да | Роль сообщения: `system`, `user`, `assistant` или `tool`. |
| messages[].content | string/array | Да | Содержимое сообщения. Для текста используйте string, для мультимодального ввода — array на поддерживаемых моделях. |
| temperature | number | Нет | Случайность генерации, обычно от `0` до `2`. |
| top_p | number | Нет | Nucleus sampling. Не рекомендуется сильно менять его одновременно с `temperature`. |
| max_tokens | integer | Нет | Максимум генерируемых token. В примере `4096`; реальный лимит зависит от модели и upstream. |
| stream | boolean | Нет | Для обычного ответа не передавайте или укажите `false`. |
| stop | string/array | Нет | Останавливает генерацию при появлении указанной последовательности. |
| tools | array | Нет | Описания функций или инструментов, если модель поддерживает. |
| tool_choice | string/object | Нет | Управляет выбором инструмента: `auto`, `none` или конкретный инструмент. |
| response_format | object | Нет | Запрашивает формат ответа, например JSON object или JSON Schema. |
| presence_penalty | number | Нет | Штрафует повторение тем, обычно от `-2` до `2`. |
| frequency_penalty | number | Нет | Штрафует повторение формулировок, обычно от `-2` до `2`. |
| user | string | Нет | Идентификатор конечного пользователя для аудита и risk control. |

## Пример запроса

```bash
curl https://aivrae.com/v1/chat/completions \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-5.4-mini",
    "messages": [
      { "role": "system", "content": "You are a concise assistant." },
      { "role": "user", "content": "Write a short welcome message." }
    ],
    "temperature": 0.7,
    "max_tokens": 4096
  }'
```

## Поля ответа

| Поле | Описание |
| --- | --- |
| id | Идентификатор ответа. |
| object | Тип объекта, обычно `chat.completion`. |
| created | Unix timestamp создания. |
| model | Модель, которая сформировала ответ. |
| choices[].message.role | Возвращенная роль, обычно `assistant`. |
| choices[].message.content | Основной сгенерированный текст. |
| choices[].finish_reason | Причина завершения: `stop`, `length`, `tool_calls`. `length` означает достижение лимита вывода. |
| usage.prompt_tokens | Количество входных token. |
| usage.completion_tokens | Количество выходных token. |
| usage.total_tokens | Общее количество token, полезно для проверки списаний. |

## Официальная документация

- [OpenAI Chat Completions API](https://platform.openai.com/docs/api-reference/chat/create)
- [OpenAI Text generation guide](https://platform.openai.com/docs/guides/text-generation)
