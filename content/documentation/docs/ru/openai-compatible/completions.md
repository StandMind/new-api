Completions — устаревший формат текстового дополнения. Новым проектам лучше использовать Chat Completions, если старый клиент или модель не требуют другого.

**Endpoint:** `POST /v1/completions`

```
curl https://aivrae.com/v1/completions \
  -H "Authorization: Bearer YOUR_AIVRAE_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "MODEL_NAME",
    "prompt": "Напиши короткий слоган для API-шлюза.",
    "max_tokens": 80
  }'
```
