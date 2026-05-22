Tên mô hình phải khớp chính xác với console. Mô hình khả dụng có thể khác nhau theo tài khoản, nhóm hoặc gói; một số mô hình chỉ hỗ trợ endpoint hoặc tham số cụ thể.

**Endpoint:** `GET /v1/models`

| Trường / khái niệm | Mô tả |
| --- | --- |
| id | Giá trị truyền vào trường model. |
| owned_by / provider | Nhà cung cấp hoặc loại route nếu có. |
| Độ dài ngữ cảnh | Theo console; quá dài có thể trả 400, 413 hoặc lỗi dịch vụ. |
| Tương thích endpoint | Chat dùng chat completions, embedding dùng embeddings, Claude dùng /v1/messages, Gemini dùng generateContent. |

```
curl https://aivrae.com/v1/models \
  -H "Authorization: Bearer YOUR_AIVRAE_API_KEY"
```
