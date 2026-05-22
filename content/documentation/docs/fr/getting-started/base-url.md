La plupart des clients demandent uniquement la Base URL. Préférez /v1 afin que les SDK ajoutent les chemins spécifiques.

| Cas | Valeur recommandée | Notes |
| --- | --- | --- |
| SDK OpenAI / clients courants | https://aivrae.com/v1 | Recommandé. |
| Endpoint chat complet | https://aivrae.com/v1/chat/completions | Seulement si le client l'exige. |
| Liste des modèles | https://aivrae.com/v1/models | Vérifie les modèles visibles. |
| Claude natif | https://aivrae.com/v1/messages | Requiert x-api-key et anthropic-version. |
| Gemini natif | https://aivrae.com/v1beta/models/{model}:generateContent | Remplacez {model}. |

```
Base URL recommandée
https://aivrae.com/v1

Endpoints complets courants
POST https://aivrae.com/v1/chat/completions
POST https://aivrae.com/v1/responses
POST https://aivrae.com/v1/completions
POST https://aivrae.com/v1/embeddings
GET  https://aivrae.com/v1/models

Endpoints texte natifs
POST https://aivrae.com/v1/messages
POST https://aivrae.com/v1beta/models/{model}:generateContent
```
