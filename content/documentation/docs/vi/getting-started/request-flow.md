API văn bản Aivrae dùng yêu cầu HTTPS JSON. Cần có xác thực, tên mô hình và nội dung đầu vào.

| Mục | Giá trị | Ghi chú |
| --- | --- | --- |
| Xác thực | Authorization: Bearer YOUR_AIVRAE_API_KEY | Dùng cho endpoint OpenAI compatible và Gemini native. |
| Xác thực Claude | x-api-key: YOUR_AIVRAE_API_KEY | Dùng cho Claude native. |
| Content type | Content-Type: application/json | Endpoint văn bản dùng JSON. |
| Tên mô hình | Sao chép từ console | Phân biệt hoa thường và hậu tố. |
| Timeout | Từ 60 giây | Streaming cần giữ kết nối. |
| Nhật ký | Thời gian, mô hình, trạng thái, lỗi | Quan trọng cho thanh toán và xử lý lỗi. |
