Responses là định dạng vào/ra mới hơn. Một số mô hình và công cụ mới có thể ưu tiên định dạng này. Nếu client chưa hỗ trợ, tiếp tục dùng Chat Completions.

**Endpoint:** `POST /v1/responses`

| Mục | Chat Completions | Responses |
| --- | --- | --- |
| Trường nhập | messages | input |
| Cách dùng | Client chat truyền thống | Mô hình mới và workflow thống nhất |
| Streaming | stream: true | stream nếu hỗ trợ |
| Khuyến nghị | Mặc định | Khi mô hình hoặc công cụ yêu cầu |

```
curl https://aivrae.com/v1/responses \
  -H "Authorization: Bearer YOUR_AIVRAE_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "MODEL_NAME",
    "input": [
      { "role": "user", "content": "Summarize the benefits of API gateways." }
    ]
  }'
```
