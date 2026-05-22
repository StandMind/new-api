Function calling và đầu ra có cấu trúc vẫn là API văn bản, nhưng phụ thuộc vào năng lực mô hình và route. Hãy chuẩn bị fallback cho mô hình không hỗ trợ.

**Endpoint:** `POST /v1/chat/completions`

## Ví dụ function calling

```
curl https://aivrae.com/v1/chat/completions \
  -H "Authorization: Bearer YOUR_AIVRAE_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "MODEL_NAME",
    "messages": [
      { "role": "user", "content": "Thời tiết ở Hà Nội hôm nay thế nào?" }
    ],
    "tools": [
      {
        "type": "function",
        "function": {
          "name": "get_weather",
          "description": "Lấy thời tiết hiện tại",
          "parameters": {
            "type": "object",
            "properties": {
              "city": { "type": "string" }
            },
            "required": ["city"]
          }
        }
      }
    ],
    "tool_choice": "auto"
  }'
```

## Ví dụ JSON

```
curl https://aivrae.com/v1/chat/completions \
  -H "Authorization: Bearer YOUR_AIVRAE_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "MODEL_NAME",
    "response_format": { "type": "json_object" },
    "messages": [
      { "role": "system", "content": "Chỉ trả về JSON hợp lệ." },
      { "role": "user", "content": "Tạo một đối tượng JSON có name và slogan." }
    ]
  }'
```
