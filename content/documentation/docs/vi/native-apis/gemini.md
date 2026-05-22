Nếu Gemini native được bật, hãy dùng yêu cầu kiểu Google Gemini generateContent. Định dạng này thường chỉ dùng cho mô hình Gemini. Để tương thích rộng, Chat Completions vẫn là mặc định khuyến nghị.

**Endpoint:** `POST /v1beta/models/{model}:generateContent`

| Header / tham số | Mô tả |
| --- | --- |
| Authorization | `Bearer YOUR_AIVRAE_API_KEY`. |
| model | Trong đường dẫn URL, ví dụ `gemini-2.5-flash`. |
| contents | Mảng đầu vào Gemini gồm role và parts. |
| generationConfig | Tham số sinh như temperature và maxOutputTokens. |

```
curl https://aivrae.com/v1beta/models/GEMINI_MODEL_NAME:generateContent \
  -H "Authorization: Bearer YOUR_AIVRAE_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "contents": [
      {
        "role": "user",
        "parts": [
          { "text": "Viết một phần giới thiệu dự án ngắn gọn." }
        ]
      }
    ],
    "generationConfig": {
      "temperature": 0.7,
      "maxOutputTokens": 512
    }
  }'
```
