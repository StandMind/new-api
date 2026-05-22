Yêu cầu streaming trả Server-Sent Events theo từng phần. Phù hợp cho giao diện chat, CLI và câu trả lời dài.

**Endpoint:** `POST /v1/chat/completions`

- Đọc dòng data và nối delta.content vào câu trả lời.
- data: [DONE] là kết thúc.
- include_usage có thể trả usage ở cuối nếu hỗ trợ.
- Không retry vô hạn một yêu cầu bị ngắt.

```
curl https://aivrae.com/v1/chat/completions \
  -H "Authorization: Bearer YOUR_AIVRAE_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "MODEL_NAME",
    "stream": true,
    "stream_options": { "include_usage": true },
    "messages": [
      { "role": "user", "content": "Gợi ý ba tên sản phẩm." }
    ]
  }'
```
