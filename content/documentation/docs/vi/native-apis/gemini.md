API native Gemini dùng định dạng Google Gemini `generateContent`. Nó hữu ích khi client đã dùng SDK Gemini hoặc cần các trường native như `contents`, `generationConfig`, `safetySettings`. Nếu cần tương thích rộng với client, nên dùng Chat Completions tương thích OpenAI.

Endpoint: `POST /v1beta/models/{model}:generateContent`

> [!NOTE]
> `{model}` là một phần của URL, ví dụ `gemini-2.5-flash`. Mô hình và giá khả dụng phụ thuộc cấu hình trong console.

## Header

| Tham số | Kiểu | Bắt buộc | Mô tả |
| --- | --- | --- | --- |
| Authorization | string | Có | Dùng `Bearer YOUR_API_KEY`. |
| Content-Type | string | Có | Phải là `application/json`. |

## Tham số đường dẫn

| Tham số | Kiểu | Bắt buộc | Mô tả |
| --- | --- | --- | --- |
| model | string | Có | Tên mô hình Gemini, ví dụ `gemini-2.5-flash`. Tester đồng bộ ô model vào đoạn `{model}` trong URL. |

## Body yêu cầu

| Tham số | Kiểu | Bắt buộc | Mô tả |
| --- | --- | --- | --- |
| contents | array | Có | Mảng nội dung đầu vào. Với hội thoại nhiều lượt, thêm nhiều phần tử theo thứ tự. |
| contents[].role | string | Không | Vai trò, thường là `user` hoặc `model`. |
| contents[].parts | array | Có | Các phần nội dung: văn bản, ảnh, file. |
| contents[].parts[].text | string | Không | Nội dung văn bản. |
| systemInstruction | object | Không | Chỉ dẫn hệ thống để định hướng hành vi mô hình. |
| generationConfig | object | Không | Cấu hình sinh. |
| generationConfig.temperature | number | Không | Điều khiển độ ngẫu nhiên. |
| generationConfig.topP | number | Không | Nucleus sampling. |
| generationConfig.topK | integer | Không | Lấy mẫu trong K token có xác suất cao nhất. |
| generationConfig.maxOutputTokens | integer | Không | Số token đầu ra tối đa. Ví dụ dùng `4096`. |
| generationConfig.stopSequences | array | Không | Dừng sinh khi gặp chuỗi chỉ định. |
| safetySettings | array | Không | Cấu hình an toàn, tùy upstream hỗ trợ. |
| tools | array | Không | Định nghĩa tool native Gemini, ví dụ function calling. |
| toolConfig | object | Không | Cấu hình gọi tool. |

## Ví dụ

```bash
curl https://aivrae.com/v1beta/models/gemini-2.5-flash:generateContent \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "contents": [
      {
        "role": "user",
        "parts": [
          { "text": "Write a concise project summary." }
        ]
      }
    ],
    "generationConfig": {
      "temperature": 0.7,
      "maxOutputTokens": 4096
    }
  }'
```

## Trường phản hồi

| Trường | Mô tả |
| --- | --- |
| candidates | Mảng phản hồi ứng viên. Hầu hết client đọc ứng viên đầu tiên. |
| candidates[].content.parts[].text | Văn bản được sinh. |
| candidates[].finishReason | Lý do kết thúc: `STOP`, `MAX_TOKENS`, `SAFETY`. |
| candidates[].safetyRatings | Chi tiết đánh giá an toàn. |
| usageMetadata.promptTokenCount | Token đầu vào. |
| usageMetadata.candidatesTokenCount | Token đầu ra. |
| usageMetadata.totalTokenCount | Tổng token. |

## Tài liệu chính thức

- [Gemini generateContent API](https://ai.google.dev/api/generate-content)
- [Gemini function calling](https://ai.google.dev/gemini-api/docs/function-calling)
