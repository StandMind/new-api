Нативный API Gemini использует формат Google Gemini `generateContent`. Он полезен, если клиент уже использует Gemini SDK или нужны нативные поля `contents`, `generationConfig`, `safetySettings`. Для широкой совместимости клиентов лучше использовать OpenAI-compatible Chat Completions.

Endpoint: `POST /v1beta/models/{model}:generateContent`

> [!NOTE]
> `{model}` является частью URL, например `gemini-2.5-flash`. Доступные модели и цены зависят от настроек консоли.

## Заголовки

| Параметр | Тип | Обязателен | Описание |
| --- | --- | --- | --- |
| Authorization | string | Да | Используйте `Bearer YOUR_API_KEY`. |
| Content-Type | string | Да | Должен быть `application/json`. |

## Параметры пути

| Параметр | Тип | Обязателен | Описание |
| --- | --- | --- | --- |
| model | string | Да | Имя модели Gemini, например `gemini-2.5-flash`. Тестер синхронизирует поле модели с сегментом `{model}` в URL. |

## Тело запроса

| Параметр | Тип | Обязателен | Описание |
| --- | --- | --- | --- |
| contents | array | Да | Массив входного содержимого. Для multi-turn добавляйте элементы по порядку. |
| contents[].role | string | Нет | Роль, обычно `user` или `model`. |
| contents[].parts | array | Да | Части содержимого: текст, изображения, файлы. |
| contents[].parts[].text | string | Нет | Текстовый ввод. |
| systemInstruction | object | Нет | Системная инструкция, задающая поведение модели. |
| generationConfig | object | Нет | Настройки генерации. |
| generationConfig.temperature | number | Нет | Управляет случайностью. |
| generationConfig.topP | number | Нет | Nucleus sampling. |
| generationConfig.topK | integer | Нет | Выбор среди K наиболее вероятных token. |
| generationConfig.maxOutputTokens | integer | Нет | Максимум выходных token. В примере `4096`. |
| generationConfig.stopSequences | array | Нет | Останавливает генерацию при появлении последовательности. |
| safetySettings | array | Нет | Настройки безопасности, если upstream поддерживает. |
| tools | array | Нет | Нативные инструменты Gemini, например function calling. |
| toolConfig | object | Нет | Конфигурация вызова инструментов. |

## Пример запроса

```bash
curl https://aivrae.com/v1beta/models/gemini-2.5-flash:generateContent \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "contents": [
      {
        "role": "user",
        "parts": [
          { "text": "Write a concise project summary." }
        ]
      }
    ],
    "generationConfig": {
      "temperature": 0.7,
      "maxOutputTokens": 4096
    }
  }'
```

## Поля ответа

| Поле | Описание |
| --- | --- |
| candidates | Массив кандидатов ответа. Обычно клиент читает первый. |
| candidates[].content.parts[].text | Сгенерированный текст. |
| candidates[].finishReason | Причина завершения: `STOP`, `MAX_TOKENS`, `SAFETY`. |
| candidates[].safetyRatings | Детали оценки безопасности. |
| usageMetadata.promptTokenCount | Входные token. |
| usageMetadata.candidatesTokenCount | Выходные token. |
| usageMetadata.totalTokenCount | Всего token. |

## Официальная документация

- [Gemini generateContent API](https://ai.google.dev/api/generate-content)
- [Gemini function calling](https://ai.google.dev/gemini-api/docs/function-calling)
