Aivrae recommends OpenAI-compatible Chat Completions as the default request format. It works best for most text chat models, SDKs, clients, and automation tools. Some model families also support native formats such as Claude Messages and Gemini generateContent.

| Format | Model range | Endpoint | Authentication | Recommended use |
| --- | --- | --- | --- | --- |
| OpenAI-compatible | Most text chat models visible in the console | `/v1/chat/completions` | `Authorization: Bearer ...` | Default choice with the broadest client compatibility. |
| Claude native | Claude models | `/v1/messages` | `x-api-key: ...` | Claude Code, Claude native SDKs, or Anthropic Messages parameters. |
| Gemini native | Gemini models | `/v1beta/models/{model}:generateContent` | `Authorization: Bearer ...` | Gemini CLI, Gemini native SDKs, or `contents/parts` request bodies. |

## Compatibility rules

- Native formats usually apply only to their model family: Claude format for Claude models, Gemini format for Gemini models.
- OpenAI-compatible format is the recommended default, but do not assume every model supports every OpenAI parameter.
- Embedding models should use `/v1/embeddings`, not a chat endpoint.
- Availability is determined by console permissions and actual API responses.

## Selection guide

- If unsure, start with `/v1/chat/completions`.
- For general clients, choose OpenAI Compatible Provider.
- For Claude Code or Claude native SDKs, use `/v1/messages`.
- For Gemini native tooling, use `/v1beta/models/{model}:generateContent`.
