Completions est l'ancien format de complétion texte. Préférez Chat Completions sauf si un ancien client ou modèle l'exige.

**Endpoint:** `POST /v1/completions`

```
curl https://aivrae.com/v1/completions \
  -H "Authorization: Bearer YOUR_AIVRAE_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "MODEL_NAME",
    "prompt": "Écris un court slogan pour une passerelle API.",
    "max_tokens": 80
  }'
```
