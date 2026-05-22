Si l'accès Gemini natif est activé, utilisez des requêtes de style Google Gemini generateContent. Ce format s'applique généralement uniquement aux modèles Gemini. Pour la compatibilité générale, Chat Completions reste recommandé.

**Endpoint:** `POST /v1beta/models/{model}:generateContent`

| En-tête / paramètre | Description |
| --- | --- |
| Authorization | Utilisez `Bearer YOUR_AIVRAE_API_KEY`. |
| model | Dans le chemin URL, par exemple `gemini-2.5-flash`. |
| contents | Tableau d'entrée Gemini avec role et parts. |
| generationConfig | Paramètres comme temperature et maxOutputTokens. |

```
curl https://aivrae.com/v1beta/models/GEMINI_MODEL_NAME:generateContent \
  -H "Authorization: Bearer YOUR_AIVRAE_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "contents": [
      {
        "role": "user",
        "parts": [
          { "text": "Rédige une présentation concise du projet." }
        ]
      }
    ],
    "generationConfig": {
      "temperature": 0.7,
      "maxOutputTokens": 512
    }
  }'
```
