Chat Completions dạng stream trả về Server-Sent Events trong khi mô hình đang sinh. Chế độ này phù hợp cho giao diện chat, terminal thời gian thực, câu trả lời dài và trường hợp cần thấy token đầu tiên nhanh.

Endpoint: `POST /v1/chat/completions`

> [!TIP]
> API tester ở cuối trang hiện đã đọc được phản hồi stream. Đặt `"stream": true` trong body, các chunk sẽ xuất hiện trong khung phản hồi khi đến.

## Body yêu cầu

| Tham số | Kiểu | Bắt buộc | Mô tả |
| --- | --- | --- | --- |
| model | string | Có | Tên mô hình, ví dụ `gpt-5.4-mini`. |
| messages | array | Có | Tin nhắn hội thoại theo thứ tự, cùng cấu trúc với chế độ không stream. |
| stream | boolean | Có | Phải là `true` để bật stream. |
| stream_options | object | Không | Tùy chọn bổ sung cho stream. |
| stream_options.include_usage | boolean | Không | Nếu `true`, upstream hỗ trợ có thể gửi usage gần cuối stream. |
| temperature | number | Không | Điều khiển độ ngẫu nhiên. |
| top_p | number | Không | Nucleus sampling. |
| max_tokens | integer | Không | Số token tối đa được sinh. Ví dụ dùng `4096`. |
| tools | array | Không | Định nghĩa tool. Tool call dạng stream trả qua `delta.tool_calls`. |
| tool_choice | string/object | Không | Điều khiển chọn tool. |
| response_format | object | Không | Yêu cầu định dạng đầu ra. Với JSON, cũng nên yêu cầu JSON trong prompt. |

## Định dạng SSE

| Trường | Mô tả |
| --- | --- |
| data | Dòng dữ liệu SSE, thường là JSON hoặc `[DONE]`. |
| choices[].delta.content | Phần văn bản mới của chunk này, cần nối vào câu trả lời hiện tại. |
| choices[].delta.role | Vai trò có thể xuất hiện ở đầu stream. |
| choices[].delta.tool_calls | Mảnh tool call tăng dần, gộp theo `index`. |
| choices[].finish_reason | Lý do kết thúc. Nếu không rỗng thì choice đó đã xong. |
| usage | Có thể xuất hiện gần cuối khi `stream_options.include_usage=true` và upstream hỗ trợ. |

## Ví dụ

```bash
curl https://aivrae.com/v1/chat/completions \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-5.4-mini",
    "stream": true,
    "stream_options": { "include_usage": true },
    "messages": [
      { "role": "user", "content": "Give me three product name ideas." }
    ],
    "max_tokens": 4096
  }'
```

## Stream ví dụ

```text
data: {"choices":[{"delta":{"role":"assistant"},"index":0}]}

data: {"choices":[{"delta":{"content":"Ý tưởng"},"index":0}]}

data: {"choices":[{"delta":{"content":" đầu tiên"},"index":0}]}

data: [DONE]
```

## Ghi chú client

- Đọc từng dòng `data:` và nối `delta.content` vào câu trả lời hiện tại.
- Đóng reader khi nhận `data: [DONE]`.
- Không retry vô hạn cùng một yêu cầu sau lỗi mạng để tránh sinh trùng và tính phí trùng.
- Nếu `finish_reason` là `length`, tăng `max_tokens` hoặc rút ngắn prompt.

## Tài liệu chính thức

- [OpenAI Chat Completions API](https://platform.openai.com/docs/api-reference/chat/create)
- [OpenAI Streaming guide](https://platform.openai.com/docs/guides/streaming-responses)
