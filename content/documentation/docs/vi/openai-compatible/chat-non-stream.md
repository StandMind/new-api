Yêu cầu không stream trả toàn bộ kết quả sau khi sinh xong. Phù hợp cho văn bản ngắn, tác vụ nền, phân loại và tóm tắt.

**Endpoint:** `POST /v1/chat/completions`

| Tham số | Bắt buộc | Mô tả |
| --- | --- | --- |
| model | Có | Tên mô hình từ console. |
| messages | Có | Mảng hội thoại với system, user, assistant. |
| temperature | Không | Từ 0 đến 2; thường 0.2-0.8. |
| max_tokens | Không | Giới hạn độ dài đầu ra. |
| stream | Không | Bỏ qua hoặc false cho không stream. |

```
curl https://aivrae.com/v1/chat/completions \
  -H "Authorization: Bearer YOUR_AIVRAE_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "MODEL_NAME",
    "messages": [
      { "role": "system", "content": "Bạn là trợ lý trả lời ngắn gọn." },
      { "role": "user", "content": "Viết một lời chào ngắn." }
    ],
    "temperature": 0.7,
    "max_tokens": 512
  }'
```

## Trường phản hồi thường gặp

| Trường | Mô tả |
| --- | --- |
| choices[0].message.content | Nội dung văn bản chính do mô hình trả về. |
| choices[0].finish_reason | stop, length, tool_calls hoặc lý do tương tự. |
| usage.prompt_tokens | Số token đầu vào. |
| usage.completion_tokens | Số token đầu ra. |
| usage.total_tokens | Tổng số token, thường dùng làm tham chiếu tính phí. |
