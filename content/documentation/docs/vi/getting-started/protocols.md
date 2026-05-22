Aivrae khuyến nghị Chat Completions tương thích OpenAI làm định dạng mặc định. Một số họ mô hình cũng hỗ trợ định dạng native như Claude Messages và Gemini generateContent.

| Định dạng | Mô hình | Endpoint | Xác thực | Khi nên dùng |
| --- | --- | --- | --- | --- |
| OpenAI-compatible | Hầu hết mô hình chat văn bản trong console | `/v1/chat/completions` | `Authorization: Bearer ...` | Lựa chọn mặc định. |
| Claude native | Mô hình Claude | `/v1/messages` | `x-api-key: ...` | Claude Code hoặc SDK Claude native. |
| Gemini native | Mô hình Gemini | `/v1beta/models/{model}:generateContent` | `Authorization: Bearer ...` | Công cụ Gemini native. |

## Quy tắc tương thích

- Định dạng native thường chỉ áp dụng cho đúng họ mô hình.
- OpenAI-compatible là lựa chọn khuyến nghị, nhưng không phải mọi mô hình đều hỗ trợ mọi tham số OpenAI.
- Embeddings dùng `/v1/embeddings`.
- Khả dụng phụ thuộc quyền trong console và phản hồi API thực tế.

## Gợi ý lựa chọn

- Nếu chưa chắc, bắt đầu với `/v1/chat/completions`.
- Với client phổ thông, chọn OpenAI Compatible Provider.
- Với Claude Code hoặc SDK Claude native, dùng `/v1/messages`.
- Với công cụ Gemini native, dùng `/v1beta/models/{model}:generateContent`.
