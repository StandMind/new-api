Completions is the legacy text-completion format. New projects should prefer Chat Completions unless an old client or specific model requires it.

**Endpoint:** `POST /v1/completions`

```
curl https://aivrae.com/v1/completions \
  -H "Authorization: Bearer YOUR_AIVRAE_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "MODEL_NAME",
    "prompt": "Write a short tagline for an API gateway.",
    "max_tokens": 80
  }'
```
