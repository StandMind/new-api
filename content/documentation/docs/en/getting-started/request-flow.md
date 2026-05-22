Aivrae text APIs use HTTPS JSON requests. Include authentication, model name, and input content. The response shape depends on the selected compatible protocol.

| Item | Value | Notes |
| --- | --- | --- |
| Authentication | Authorization: Bearer YOUR_AIVRAE_API_KEY | OpenAI-compatible endpoints and Gemini native format use Bearer auth. |
| Claude authentication | x-api-key: YOUR_AIVRAE_API_KEY | Claude native format uses Anthropic-style headers. |
| Content type | Content-Type: application/json | Text endpoints use JSON. |
| Model name | Copy from the console | Do not guess model names; case and suffixes matter. |
| Timeout | 60 seconds or more | Streaming clients must keep reading the connection. |
| Logs | Time, model, status, error message | Important for billing and service troubleshooting. |
