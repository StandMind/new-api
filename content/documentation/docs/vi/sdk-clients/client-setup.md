Hầu hết client bên thứ ba dùng OpenAI Compatible Provider. Tên trường có thể khác nhau, nhưng giá trị chính là API Key, Base URL và Model.

| Client | Loại provider | Base URL | Ghi chú |
| --- | --- | --- | --- |
| Cherry Studio | OpenAI Compatible | https://aivrae.com/v1 | Sao chép mô hình từ console. |
| ChatBox | OpenAI API Compatible | https://aivrae.com/v1 | Test văn bản ngắn trước. |
| Cursor / Cline | OpenAI Compatible / Custom Provider | https://aivrae.com/v1 | Hữu ích cho code và agent. |
| Dify | OpenAI-API-compatible | https://aivrae.com/v1 | Thêm chat và embedding riêng. |
| LobeChat / NextChat | Custom OpenAI endpoint | https://aivrae.com/v1 | Chỉ dùng endpoint đầy đủ nếu cần. |
| Aider / Claude Code | OpenAI compatible hoặc native | https://aivrae.com/v1 | Claude: /v1/messages; Gemini: /v1beta/models/{model}:generateContent. |
