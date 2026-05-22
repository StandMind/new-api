Ứng dụng nên xử lý rõ các mã trạng thái phổ biến và ghi log lỗi ở backend.

| Trạng thái | Ý nghĩa | Cách xử lý |
| --- | --- | --- |
| 400 | Body, tham số hoặc JSON không hợp lệ | Kiểm tra body, endpoint và tham số. |
| 401 | Thiếu/sai/hết hạn/vô hiệu key | Sao chép lại key và kiểm tra header. |
| 403 | Không có quyền với mô hình, nhóm hoặc tính năng | Kiểm tra quyền, gói và nhóm. |
| 404 | Sai đường dẫn hoặc mô hình | Kiểm tra Base URL, native path và model. |
| 413 | Body hoặc context quá lớn | Giảm messages, contents hoặc input. |
| 429 | Rate, concurrency, quota hoặc số dư | Giảm song song và kiểm tra số dư. |
| 500 / 502 / 503 | Lỗi tạm thời từ Aivrae hoặc dịch vụ mô hình | Thử lại sau; liên hệ hỗ trợ nếu kéo dài. |
