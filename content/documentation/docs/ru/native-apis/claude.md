Нативный API Claude использует протокол Anthropic Messages. Он полезен, если клиент уже использует Claude SDK или нужны нативные поля `system`, `tools`, `thinking`, `stream`. Для широкой совместимости клиентов лучше использовать OpenAI-compatible Chat Completions.

Endpoint: `POST /v1/messages`

> [!NOTE]
> Нативные запросы Claude используют заголовок `x-api-key`, а не `Authorization: Bearer ...`. Тестер автоматически устанавливает подходящий заголовок для этого endpoint.

## Заголовки

| Параметр | Тип | Обязателен | Описание |
| --- | --- | --- | --- |
| x-api-key | string | Да | Ваш API Key. |
| anthropic-version | string | Да | Версия Anthropic API. В примере `2023-06-01`. |
| Content-Type | string | Да | Должен быть `application/json`. |

## Тело запроса

| Параметр | Тип | Обязателен | Описание |
| --- | --- | --- | --- |
| model | string | Да | Имя модели Claude, включенной в консоли. |
| max_tokens | integer | Да | Максимум выходных token. В примере `4096`. |
| messages | array | Да | Сообщения диалога. |
| messages[].role | string | Да | Роль сообщения, обычно `user` или `assistant`. |
| messages[].content | string/array | Да | Содержимое: string для текста или блоки контента для мультимодальности. |
| system | string/array | Нет | Системный prompt, задающий поведение ассистента. |
| temperature | number | Нет | Управляет случайностью. |
| top_p | number | Нет | Nucleus sampling. |
| top_k | integer | Нет | Ограничивает кандидатные token. |
| stop_sequences | array | Нет | Останавливает генерацию при появлении последовательности. |
| stream | boolean | Нет | `true` для нативного streaming Claude. |
| tools | array | Нет | Описания инструментов. |
| tool_choice | object | Нет | Управляет выбором инструмента. |
| metadata | object | Нет | Дополнительные пользовательские или бизнес-метаданные. |
| thinking | object | Нет | Настройки extended thinking, применяются только на поддерживаемых моделях. |

## Пример запроса

```bash
curl https://aivrae.com/v1/messages \
  -H "x-api-key: YOUR_API_KEY" \
  -H "anthropic-version: 2023-06-01" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "claude-3-5-sonnet-20241022",
    "max_tokens": 4096,
    "messages": [
      { "role": "user", "content": "Write a concise project summary." }
    ]
  }'
```

## Поля ответа

| Поле | Описание |
| --- | --- |
| id | Идентификатор ответа Claude. |
| type | Тип объекта, обычно `message`. |
| role | Возвращенная роль, обычно `assistant`. |
| content | Массив блоков контента. Текст обычно в `content[].text`. |
| model | Модель, которая сформировала ответ. |
| stop_reason | Причина завершения: `end_turn`, `max_tokens`, `tool_use`. |
| usage.input_tokens | Входные token. |
| usage.output_tokens | Выходные token. |

## Официальная документация

- [Anthropic Messages API](https://docs.anthropic.com/en/api/messages)
- [Anthropic streaming Messages](https://docs.anthropic.com/en/api/messages-streaming)
