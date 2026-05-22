Completions 是较旧的文本补全格式。新项目优先使用 Chat Completions；只有旧 SDK、旧插件或特定模型要求时再使用。

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
