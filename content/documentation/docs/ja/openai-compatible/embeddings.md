Embeddings はテキストをベクトルに変換し、RAG、意味検索、類似度、クラスタリングに使います。自然言語の回答は返しません。

**Endpoint:** `POST /v1/embeddings`

```
curl https://aivrae.com/v1/embeddings \
  -H "Authorization: Bearer YOUR_AIVRAE_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "EMBEDDING_MODEL_NAME",
    "input": "ベクトル化するテキスト"
  }'
```
