## Platform overview

Aivrae is an AI API aggregation gateway for global developers and teams. It provides a unified API endpoint, API key management, model access, prepaid usage accounting, and request logs. You can use OpenAI-compatible formats for text chat, text generation, embeddings, the Responses API, Claude native messages, and Gemini native generateContent without maintaining separate integration code for every model service.

This public documentation focuses on text services. Available features are determined by the console, API responses, and official Aivrae support replies.

## Core capabilities

- Unified access: use `https://aivrae.com/v1` as the OpenAI-compatible Base URL and authenticate with an API key created in the console.
- Protocol compatibility: OpenAI-compatible format is the recommended default, with examples for Claude Messages and Gemini generateContent native formats.
- Model switching: use model names shown in your console and switch models by changing the `model` field or the native endpoint path.
- Usage visibility: check balance, request history, model usage, and error states in the console.
- Built-in testing: API documentation pages include default request values so you can test each endpoint after entering an API key.

## Recommended integration flow

1. Sign in to the Aivrae console.
2. Create an API key and copy an available model name from the console.
3. Send a small test request from the built-in tester in this documentation.
4. Set your application or client Base URL to `https://aivrae.com/v1`.
5. In production, log request IDs, status codes, model names, and error messages for troubleshooting and usage reconciliation.

## Usage comparison

| Item | Using Aivrae | Integrating model services separately |
| --- | --- | --- |
| Endpoint | One OpenAI-compatible Base URL plus selected native text endpoints | Multiple service endpoints |
| API key | Managed in one console | Managed separately per service |
| Client compatibility | Works with most OpenAI-compatible clients | Requires service-specific setup |
| Model switching | Test available models by changing the model name | Often requires endpoint and parameter changes |
| Usage records | Centralized in the console | Spread across separate service dashboards |
| Documentation testing | Test the current endpoint directly on the page | Usually requires Postman or custom code |

## Current documentation scope

| Capability | Status | Notes |
| --- | --- | --- |
| Text chat | Supported | Use `/v1/chat/completions`. |
| Streaming | Supported | Set `stream: true` and read SSE deltas. |
| Function calling / JSON output | Supported for text workflows | Depends on whether the selected model supports `tools` and `response_format`. |
| Responses API | Supported | Use `/v1/responses` for newer OpenAI-compatible workflows. |
| Embeddings | Supported | For RAG, semantic search, similarity, and clustering. |
| Claude native text format | Model-dependent | Use `/v1/messages`, usually only with Claude models. |
| Gemini native text format | Model-dependent | Use `/v1beta/models/{model}:generateContent`, usually only with Gemini models. |
