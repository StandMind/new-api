## Tổng quan nền tảng

Aivrae là cổng tổng hợp AI API cho nhà phát triển và nhóm trên toàn cầu. Dịch vụ cung cấp endpoint thống nhất, quản lý API key, truy cập mô hình, tính toán tín dụng trả trước và nhật ký yêu cầu. Bạn có thể dùng định dạng tương thích OpenAI, Responses API, embeddings, định dạng native Claude và định dạng native Gemini.

Tài liệu công khai này tập trung vào dịch vụ văn bản. Tính năng khả dụng phụ thuộc vào bảng điều khiển, phản hồi API và hỗ trợ chính thức của Aivrae.

## Năng lực chính

- Truy cập thống nhất: dùng `https://aivrae.com/v1` làm Base URL tương thích OpenAI.
- Tương thích giao thức: khuyến nghị định dạng OpenAI compatible, đồng thời có ví dụ Claude Messages và Gemini generateContent.
- Đổi mô hình: dùng tên mô hình hiển thị trong console.
- Theo dõi sử dụng: xem số dư, lịch sử, tiêu hao và lỗi trong console.
- Kiểm thử tích hợp: trang API có sẵn giá trị yêu cầu mặc định để thử trực tiếp.

## Quy trình khuyến nghị

1. Đăng nhập console Aivrae.
2. Tạo API key và sao chép tên mô hình khả dụng.
3. Gửi một yêu cầu nhỏ bằng bộ kiểm thử trong tài liệu.
4. Đặt Base URL của ứng dụng hoặc client thành `https://aivrae.com/v1`.
5. Ở môi trường production, ghi request ID, mã trạng thái, tên mô hình và lỗi.

## So sánh cách dùng

| Mục | Dùng Aivrae | Tích hợp từng dịch vụ riêng |
| --- | --- | --- |
| Endpoint | Một Base URL tương thích OpenAI cùng một số endpoint văn bản native | Cần duy trì nhiều endpoint |
| API key | Quản lý tập trung trong một console | Quản lý riêng theo từng dịch vụ |
| Tương thích client | Dùng được với đa số client OpenAI-compatible | Cần cấu hình riêng theo dịch vụ |
| Đổi mô hình | Thử mô hình khả dụng bằng cách đổi tên mô hình | Thường phải đổi endpoint và tham số |
| Nhật ký sử dụng | Tập trung trong console | Rải rác ở nhiều dashboard |
| Kiểm thử tài liệu | Test endpoint hiện tại ngay trên trang | Thường cần Postman hoặc code riêng |

## Phạm vi hiện tại

| Khả năng | Trạng thái | Ghi chú |
| --- | --- | --- |
| Chat văn bản | Hỗ trợ | Dùng `/v1/chat/completions`. |
| Streaming | Hỗ trợ | Đặt `stream: true` và đọc SSE. |
| Function calling / JSON | Tùy mô hình | Phụ thuộc `tools` và `response_format`. |
| Responses API | Hỗ trợ | Dùng `/v1/responses`. |
| Embeddings | Hỗ trợ | RAG, tìm kiếm, tương đồng. |
| Claude native | Tùy mô hình | `/v1/messages`, thường dành cho Claude. |
| Gemini native | Tùy mô hình | `/v1beta/models/{model}:generateContent`, thường dành cho Gemini. |
