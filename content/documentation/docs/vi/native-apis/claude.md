Nếu Claude native được bật, hãy dùng giao thức Anthropic Messages. Định dạng này thường chỉ dùng cho mô hình Claude. Để tương thích rộng, Chat Completions vẫn là mặc định khuyến nghị.

**Endpoint:** `POST /v1/messages`

| Header / tham số | Mô tả |
| --- | --- |
| x-api-key | API key Aivrae. |
| anthropic-version | Ví dụ 2023-06-01. |
| model | Tên mô hình Claude trong console. |
| max_tokens | Thường cần thiết. |
| messages | Định dạng Anthropic messages. |

```
curl https://aivrae.com/v1/messages \
  -H "x-api-key: YOUR_AIVRAE_API_KEY" \
  -H "anthropic-version: 2023-06-01" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "CLAUDE_MODEL_NAME",
    "max_tokens": 512,
    "messages": [
      { "role": "user", "content": "Viết một phần giới thiệu dự án ngắn gọn." }
    ]
  }'
```
