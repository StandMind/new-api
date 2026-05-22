Most clients only need the Base URL. Some clients ask for a full endpoint. Prefer /v1 so SDKs can append specific paths.

| Use case | Recommended value | Notes |
| --- | --- | --- |
| OpenAI SDK / most clients | https://aivrae.com/v1 | Recommended default. |
| Client requires full chat endpoint | https://aivrae.com/v1/chat/completions | Use only when explicitly required. |
| Model list | https://aivrae.com/v1/models | Check models visible to the key. |
| Claude native format | https://aivrae.com/v1/messages | Requires x-api-key and anthropic-version headers. |
| Gemini native format | https://aivrae.com/v1beta/models/{model}:generateContent | Replace {model} with a Gemini model visible in your console. |

```
Recommended Base URL
https://aivrae.com/v1

Common full endpoints
POST https://aivrae.com/v1/chat/completions
POST https://aivrae.com/v1/responses
POST https://aivrae.com/v1/completions
POST https://aivrae.com/v1/embeddings
GET  https://aivrae.com/v1/models

Native text endpoints
POST https://aivrae.com/v1/messages
POST https://aivrae.com/v1beta/models/{model}:generateContent
```
