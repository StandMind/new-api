Embeddings convert text into vectors for RAG, semantic search, similarity matching, and clustering. They do not return natural-language answers.

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
