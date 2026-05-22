Most third-party clients integrate through OpenAI Compatible Provider. The field names differ, but the core values are API Key, Base URL, and Model.

| Client | Provider type | Base URL | Notes |
| --- | --- | --- | --- |
| Cherry Studio | OpenAI Compatible | https://aivrae.com/v1 | Copy model names from the console. |
| ChatBox | OpenAI API Compatible | https://aivrae.com/v1 | Run a short text test first. |
| Cursor / Cline | OpenAI Compatible / Custom Provider | https://aivrae.com/v1 | Useful for coding and agent workflows. |
| Dify | OpenAI-API-compatible | https://aivrae.com/v1 | Add chat and embedding models separately. |
| LobeChat / NextChat | Custom OpenAI endpoint | https://aivrae.com/v1 | Use full chat endpoint only if required. |
| Aider / Claude Code | OpenAI compatible or native protocol | https://aivrae.com/v1 | Claude native mode uses /v1/messages; Gemini native tooling uses /v1beta/models/{model}:generateContent. |
