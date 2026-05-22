Hầu hết client chỉ cần Base URL. Nên dùng /v1 để SDK tự ghép đường dẫn cụ thể.

| Trường hợp | Giá trị khuyến nghị | Ghi chú |
| --- | --- | --- |
| OpenAI SDK / đa số client | https://aivrae.com/v1 | Mặc định khuyến nghị. |
| Yêu cầu endpoint chat đầy đủ | https://aivrae.com/v1/chat/completions | Chỉ dùng khi client yêu cầu. |
| Danh sách mô hình | https://aivrae.com/v1/models | Kiểm tra mô hình key nhìn thấy. |
| Claude native | https://aivrae.com/v1/messages | Cần x-api-key và anthropic-version. |
| Gemini native | https://aivrae.com/v1beta/models/{model}:generateContent | Thay {model}. |

```
Base URL khuyến nghị
https://aivrae.com/v1

Endpoint đầy đủ thường dùng
POST https://aivrae.com/v1/chat/completions
POST https://aivrae.com/v1/responses
POST https://aivrae.com/v1/completions
POST https://aivrae.com/v1/embeddings
GET  https://aivrae.com/v1/models

Endpoint văn bản native
POST https://aivrae.com/v1/messages
POST https://aivrae.com/v1beta/models/{model}:generateContent
```
