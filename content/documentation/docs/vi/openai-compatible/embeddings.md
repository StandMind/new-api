Embeddings chuyển văn bản thành vector cho RAG, tìm kiếm ngữ nghĩa, so khớp tương đồng và phân cụm. Endpoint này không trả lời bằng ngôn ngữ tự nhiên.

**Endpoint:** `POST /v1/embeddings`

```
curl https://aivrae.com/v1/embeddings \
  -H "Authorization: Bearer YOUR_AIVRAE_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "EMBEDDING_MODEL_NAME",
    "input": "Văn bản cần vector hóa"
  }'
```
