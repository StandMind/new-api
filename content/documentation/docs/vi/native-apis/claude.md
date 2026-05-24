API native Claude dùng giao thức Anthropic Messages. Nó hữu ích khi client đã dùng SDK Claude hoặc cần các trường native như `system`, `tools`, `thinking`, `stream`. Nếu cần tương thích rộng với client, nên dùng Chat Completions tương thích OpenAI.

Endpoint: `POST /v1/messages`

> [!NOTE]
> Yêu cầu native Claude dùng header `x-api-key` thay vì `Authorization: Bearer ...`. Tester sẽ đặt header xác thực phù hợp cho endpoint này.

## Header

| Tham số | Kiểu | Bắt buộc | Mô tả |
| --- | --- | --- | --- |
| x-api-key | string | Có | API Key của bạn. |
| anthropic-version | string | Có | Phiên bản Anthropic API. Ví dụ dùng `2023-06-01`. |
| Content-Type | string | Có | Phải là `application/json`. |

## Body yêu cầu

| Tham số | Kiểu | Bắt buộc | Mô tả |
| --- | --- | --- | --- |
| model | string | Có | Tên mô hình Claude đã bật trong console. |
| max_tokens | integer | Có | Số token đầu ra tối đa. Ví dụ dùng `4096`. |
| messages | array | Có | Tin nhắn hội thoại. |
| messages[].role | string | Có | Vai trò tin nhắn, thường là `user` hoặc `assistant`. |
| messages[].content | string/array | Có | Nội dung tin nhắn. Văn bản dùng string; multimodal dùng các khối nội dung. |
| system | string/array | Không | System prompt định hướng hành vi assistant. |
| temperature | number | Không | Điều khiển độ ngẫu nhiên. |
| top_p | number | Không | Nucleus sampling. |
| top_k | integer | Không | Giới hạn token ứng viên. |
| stop_sequences | array | Không | Dừng sinh khi gặp chuỗi chỉ định. |
| stream | boolean | Không | Đặt `true` để dùng stream native Claude. |
| tools | array | Không | Định nghĩa tool. |
| tool_choice | object | Không | Điều khiển chọn tool. |
| metadata | object | Không | Metadata người dùng hoặc nghiệp vụ tùy chọn. |
| thinking | object | Không | Cấu hình extended thinking, chỉ áp dụng trên mô hình hỗ trợ. |

## Ví dụ

```bash
curl https://aivrae.com/v1/messages \
  -H "x-api-key: YOUR_API_KEY" \
  -H "anthropic-version: 2023-06-01" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "claude-3-5-sonnet-20241022",
    "max_tokens": 4096,
    "messages": [
      { "role": "user", "content": "Write a concise project summary." }
    ]
  }'
```

## Trường phản hồi

| Trường | Mô tả |
| --- | --- |
| id | ID phản hồi Claude. |
| type | Loại đối tượng, thường là `message`. |
| role | Vai trò trả về, thường là `assistant`. |
| content | Mảng khối nội dung. Văn bản thường nằm ở `content[].text`. |
| model | Mô hình tạo phản hồi. |
| stop_reason | Lý do kết thúc: `end_turn`, `max_tokens`, `tool_use`. |
| usage.input_tokens | Token đầu vào. |
| usage.output_tokens | Token đầu ra. |

## Tài liệu chính thức

- [Anthropic Messages API](https://docs.anthropic.com/en/api/messages)
- [Anthropic streaming Messages](https://docs.anthropic.com/en/api/messages-streaming)
