Si l'accès Claude natif est activé, utilisez le protocole Anthropic Messages. Ce format s'applique généralement uniquement aux modèles Claude. Pour la compatibilité générale, Chat Completions reste recommandé.

**Endpoint:** `POST /v1/messages`

| En-tête / paramètre | Description |
| --- | --- |
| x-api-key | Votre clé API Aivrae. |
| anthropic-version | Version par défaut du client, par exemple 2023-06-01. |
| model | Modèle Claude copié depuis la console. |
| max_tokens | Souvent requis. |
| messages | Format Anthropic messages. |

```
curl https://aivrae.com/v1/messages \
  -H "x-api-key: YOUR_AIVRAE_API_KEY" \
  -H "anthropic-version: 2023-06-01" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "CLAUDE_MODEL_NAME",
    "max_tokens": 512,
    "messages": [
      { "role": "user", "content": "Rédige une présentation concise du projet." }
    ]
  }'
```
