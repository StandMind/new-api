Model names must match the console exactly. Different accounts, groups, or plans may see different models; some models only support specific endpoints or parameters.

**Endpoint:** `GET /v1/models`

| Field / concept | Description |
| --- | --- |
| id | The value to pass in the model field. |
| owned_by / provider | If present, identifies provider or route type. |
| Context length | Follow the console; excessive context may return 400, 413, or service errors. |
| Endpoint compatibility | Chat models use chat completions, embedding models use embeddings, Claude native models use /v1/messages, Gemini native models use generateContent. |

```
curl https://aivrae.com/v1/models \
  -H "Authorization: Bearer YOUR_AIVRAE_API_KEY"
```
