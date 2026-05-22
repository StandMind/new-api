Aivrae по умолчанию рекомендует OpenAI-совместимый Chat Completions. Некоторые семейства моделей также поддерживают нативные форматы Claude Messages и Gemini generateContent.

| Формат | Модели | Endpoint | Auth | Когда использовать |
| --- | --- | --- | --- | --- |
| OpenAI-compatible | Большинство текстовых моделей в консоли | `/v1/chat/completions` | `Authorization: Bearer ...` | Выбор по умолчанию. |
| Claude native | Claude | `/v1/messages` | `x-api-key: ...` | Claude Code и Claude SDK. |
| Gemini native | Gemini | `/v1beta/models/{model}:generateContent` | `Authorization: Bearer ...` | Нативные инструменты Gemini. |

## Правила совместимости

- Нативные форматы обычно применяются только к своему семейству моделей.
- OpenAI-совместимый формат рекомендован, но не все параметры работают со всеми моделями.
- Embeddings вызываются через `/v1/embeddings`.
- Доступность определяется консолью и фактическим ответом API.

## Как выбрать

- Если сомневаетесь, начните с `/v1/chat/completions`.
- Для обычных клиентов выбирайте OpenAI Compatible Provider.
- Для Claude Code или нативных SDK Claude используйте `/v1/messages`.
- Для нативных инструментов Gemini используйте `/v1beta/models/{model}:generateContent`.
