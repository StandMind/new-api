Streaming Chat Completions возвращает Server-Sent Events по мере генерации ответа. Этот режим подходит для чат-интерфейсов, вывода в терминал в реальном времени, длинных ответов и сценариев, где важна задержка первого token.

Endpoint: `POST /v1/chat/completions`

> [!TIP]
> API-тестер внизу страницы теперь читает потоковые ответы. Укажите `"stream": true` в теле запроса, и фрагменты будут появляться в панели ответа по мере получения.

## Тело запроса

| Параметр | Тип | Обязателен | Описание |
| --- | --- | --- | --- |
| model | string | Да | Имя модели, например `gpt-5.4-mini`. |
| messages | array | Да | Упорядоченные сообщения диалога, структура такая же, как в обычном режиме. |
| stream | boolean | Да | Должен быть `true`, чтобы включить потоковый вывод. |
| stream_options | object | Нет | Дополнительные параметры stream. |
| stream_options.include_usage | boolean | Нет | Если `true`, поддерживаемые upstream могут отправить usage ближе к концу потока. |
| temperature | number | Нет | Управляет случайностью. |
| top_p | number | Нет | Nucleus sampling. |
| max_tokens | integer | Нет | Максимум генерируемых token. В примере `4096`. |
| tools | array | Нет | Описания инструментов. Потоковые tool calls возвращаются через `delta.tool_calls`. |
| tool_choice | string/object | Нет | Управляет выбором инструмента. |
| response_format | object | Нет | Запрашивает формат ответа. Для JSON также явно попросите JSON в prompt. |

## Формат SSE

| Поле | Описание |
| --- | --- |
| data | Строка данных SSE. Обычно это JSON или `[DONE]`. |
| choices[].delta.content | Новый текст этого фрагмента, который нужно добавить к текущему ответу. |
| choices[].delta.role | Роль, которая может прийти в начале потока. |
| choices[].delta.tool_calls | Инкрементальные фрагменты tool calls, объединяются по `index`. |
| choices[].finish_reason | Причина завершения. Непустое значение означает, что choice завершен. |
| usage | Может появиться в конце при `stream_options.include_usage=true`, если upstream поддерживает. |

## Пример запроса

```bash
curl https://aivrae.com/v1/chat/completions \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-5.4-mini",
    "stream": true,
    "stream_options": { "include_usage": true },
    "messages": [
      { "role": "user", "content": "Give me three product name ideas." }
    ],
    "max_tokens": 4096
  }'
```

## Пример потока

```text
data: {"choices":[{"delta":{"role":"assistant"},"index":0}]}

data: {"choices":[{"delta":{"content":"Первая"},"index":0}]}

data: {"choices":[{"delta":{"content":" идея"},"index":0}]}

data: [DONE]
```

## Заметки для клиента

- Читайте каждую строку `data:` и добавляйте `delta.content` к текущему ответу.
- Завершайте чтение после `data: [DONE]`.
- Не повторяйте бесконечно тот же запрос после сетевого сбоя, чтобы не получить дубли генерации и списаний.
- Если `finish_reason` равен `length`, увеличьте `max_tokens` или сократите prompt.

## Официальная документация

- [OpenAI Chat Completions API](https://platform.openai.com/docs/api-reference/chat/create)
- [OpenAI Streaming guide](https://platform.openai.com/docs/guides/streaming-responses)
