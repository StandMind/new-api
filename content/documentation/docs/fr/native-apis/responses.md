Responses est un format entrée/sortie plus récent. Certains modèles et outils peuvent le préférer. Si votre client ne le prend pas en charge, continuez avec Chat Completions.

**Endpoint:** `POST /v1/responses`

| Élément | Chat Completions | Responses |
| --- | --- | --- |
| Champ d'entrée | messages | input |
| Usage typique | Clients chat classiques | Modèles récents et workflows unifiés |
| Streaming | stream: true | stream si pris en charge |
| Recommandation | Choix par défaut | Si le modèle ou l'outil l'exige |

```
curl https://aivrae.com/v1/responses \
  -H "Authorization: Bearer YOUR_AIVRAE_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "MODEL_NAME",
    "input": [
      { "role": "user", "content": "Summarize the benefits of API gateways." }
    ]
  }'
```
