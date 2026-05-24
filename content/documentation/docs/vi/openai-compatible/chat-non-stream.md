Chat Completions không stream trả về một JSON hoàn chỉnh sau khi mô hình tạo xong. Chế độ này phù hợp cho văn bản ngắn, phân loại, tóm tắt, trích xuất có cấu trúc và tác vụ nền không cần hiển thị theo thời gian thực.

Endpoint: `POST /v1/chat/completions`

> [!NOTE]
> Định dạng yêu cầu theo phong cách OpenAI Chat Completions. Mô hình khả dụng, tham số hỗ trợ và tính phí phụ thuộc vào mô hình và kênh upstream được bật trong console.

## Header

| Tham số | Kiểu | Bắt buộc | Mô tả |
| --- | --- | --- | --- |
| Authorization | string | Có | Dùng `Bearer YOUR_API_KEY`. |
| Content-Type | string | Có | Phải là `application/json`. |

## Body yêu cầu

| Tham số | Kiểu | Bắt buộc | Mô tả |
| --- | --- | --- | --- |
| model | string | Có | Tên mô hình, ví dụ `gpt-5.4-mini`. Hãy dùng mô hình có trong danh sách hoặc console. |
| messages | array | Có | Mảng tin nhắn hội thoại theo thứ tự. |
| messages[].role | string | Có | Vai trò tin nhắn: `system`, `user`, `assistant`, `tool`. |
| messages[].content | string/array | Có | Nội dung tin nhắn. Văn bản dùng string; multimodal dùng array trên mô hình hỗ trợ. |
| temperature | number | Không | Độ ngẫu nhiên khi sinh, thường từ `0` đến `2`. |
| top_p | number | Không | Nucleus sampling. Không nên thay đổi mạnh cùng lúc với `temperature`. |
| max_tokens | integer | Không | Số token tối đa được sinh. Ví dụ dùng `4096`; giới hạn thực tế phụ thuộc mô hình và upstream. |
| stream | boolean | Không | Bỏ qua hoặc đặt `false` cho yêu cầu không stream. |
| stop | string/array | Không | Dừng sinh khi gặp chuỗi chỉ định. |
| tools | array | Không | Định nghĩa function hoặc tool, tùy mô hình hỗ trợ. |
| tool_choice | string/object | Không | Điều khiển chọn tool, ví dụ `auto`, `none` hoặc tool cụ thể. |
| response_format | object | Không | Yêu cầu định dạng đầu ra, ví dụ JSON object hoặc JSON Schema. |
| presence_penalty | number | Không | Giảm lặp chủ đề, thường từ `-2` đến `2`. |
| frequency_penalty | number | Không | Giảm lặp từ ngữ, thường từ `-2` đến `2`. |
| user | string | Không | Định danh người dùng cuối cho audit và kiểm soát rủi ro. |

## Ví dụ

```bash
curl https://aivrae.com/v1/chat/completions \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-5.4-mini",
    "messages": [
      { "role": "system", "content": "You are a concise assistant." },
      { "role": "user", "content": "Write a short welcome message." }
    ],
    "temperature": 0.7,
    "max_tokens": 4096
  }'
```

## Trường phản hồi

| Trường | Mô tả |
| --- | --- |
| id | ID phản hồi. |
| object | Loại đối tượng, thường là `chat.completion`. |
| created | Unix timestamp khi tạo phản hồi. |
| model | Mô hình tạo phản hồi. |
| choices[].message.role | Vai trò trả về, thường là `assistant`. |
| choices[].message.content | Văn bản chính được sinh. |
| choices[].finish_reason | Lý do kết thúc: `stop`, `length`, `tool_calls`. `length` nghĩa là chạm giới hạn đầu ra. |
| usage.prompt_tokens | Số token đầu vào. |
| usage.completion_tokens | Số token đầu ra. |
| usage.total_tokens | Tổng token, hữu ích để kiểm tra tính phí. |

## Tài liệu chính thức

- [OpenAI Chat Completions API](https://platform.openai.com/docs/api-reference/chat/create)
- [OpenAI Text generation guide](https://platform.openai.com/docs/guides/text-generation)
