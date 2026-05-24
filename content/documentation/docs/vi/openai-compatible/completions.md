Completions là định dạng text completion cũ. Dự án mới nên ưu tiên Chat Completions trừ khi client cũ hoặc mô hình cụ thể yêu cầu.

**Endpoint:** `POST /v1/completions`

```
curl https://aivrae.com/v1/completions \
  -H "Authorization: Bearer YOUR_AIVRAE_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "MODEL_NAME",
    "prompt": "Viết một khẩu hiệu ngắn cho cổng API.",
    "max_tokens": 4096
  }'
```
