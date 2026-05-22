Les embeddings transforment le texte en vecteurs pour RAG, recherche sémantique, similarité et clustering. Ils ne retournent pas de réponse en langage naturel.

**Endpoint:** `POST /v1/embeddings`

```
curl https://aivrae.com/v1/embeddings \
  -H "Authorization: Bearer YOUR_AIVRAE_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "EMBEDDING_MODEL_NAME",
    "input": "Texte à vectoriser"
  }'
```
