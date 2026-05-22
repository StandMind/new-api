Embeddings превращают текст в векторы для RAG, семантического поиска, сходства и кластеризации. Они не возвращают естественный язык.

**Endpoint:** `POST /v1/embeddings`

```
curl https://aivrae.com/v1/embeddings \
  -H "Authorization: Bearer YOUR_AIVRAE_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "EMBEDDING_MODEL_NAME",
    "input": "Текст для векторизации"
  }'
```
