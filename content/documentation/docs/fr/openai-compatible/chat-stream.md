Les requêtes streaming renvoient des Server-Sent Events avec du contenu incrémental. Utilisez-les pour les interfaces de chat, CLI et longues réponses.

**Endpoint:** `POST /v1/chat/completions`

- Lisez les lignes data et ajoutez delta.content à la réponse courante.
- data: [DONE] marque la fin.
- stream_options.include_usage peut renvoyer l'usage à la fin.
- Ne relancez pas indéfiniment une requête interrompue.

```
curl https://aivrae.com/v1/chat/completions \
  -H "Authorization: Bearer YOUR_AIVRAE_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "MODEL_NAME",
    "stream": true,
    "stream_options": { "include_usage": true },
    "messages": [
      { "role": "user", "content": "Donne-moi trois idées de noms de produit." }
    ]
  }'
```
