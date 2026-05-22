Function calling et sortie structurée restent des API texte, mais dépendent des capacités du modèle et de la route. Prévoyez un fallback.

**Endpoint:** `POST /v1/chat/completions`

## Exemple function calling

```
curl https://aivrae.com/v1/chat/completions \
  -H "Authorization: Bearer YOUR_AIVRAE_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "MODEL_NAME",
    "messages": [
      { "role": "user", "content": "Quel temps fait-il à Paris ?" }
    ],
    "tools": [
      {
        "type": "function",
        "function": {
          "name": "get_weather",
          "description": "Obtenir la météo actuelle",
          "parameters": {
            "type": "object",
            "properties": {
              "city": { "type": "string" }
            },
            "required": ["city"]
          }
        }
      }
    ],
    "tool_choice": "auto"
  }'
```

## Exemple JSON

```
curl https://aivrae.com/v1/chat/completions \
  -H "Authorization: Bearer YOUR_AIVRAE_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "MODEL_NAME",
    "response_format": { "type": "json_object" },
    "messages": [
      { "role": "system", "content": "Retourne uniquement du JSON valide." },
      { "role": "user", "content": "Génère un objet JSON avec name et slogan." }
    ]
  }'
```
