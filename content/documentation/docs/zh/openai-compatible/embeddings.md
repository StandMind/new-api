Embeddings 会把文本转换为向量，用于 RAG、语义检索、相似度匹配和聚类。它不会返回自然语言回答。

**Endpoint:** `POST /v1/embeddings`

```
curl https://aivrae.com/v1/embeddings \
  -H "Authorization: Bearer YOUR_AIVRAE_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "EMBEDDING_MODEL_NAME",
    "input": "Text to embed"
  }'
```
