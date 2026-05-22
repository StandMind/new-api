Aivrae recommande par défaut Chat Completions compatible OpenAI. Certains modèles prennent aussi en charge un format natif comme Claude Messages ou Gemini generateContent.

| Format | Modèles | Endpoint | Authentification | Usage conseillé |
| --- | --- | --- | --- | --- |
| OpenAI compatible | La plupart des modèles texte visibles | `/v1/chat/completions` | `Authorization: Bearer ...` | Choix par défaut. |
| Claude natif | Modèles Claude | `/v1/messages` | `x-api-key: ...` | Claude Code ou SDK Claude natif. |
| Gemini natif | Modèles Gemini | `/v1beta/models/{model}:generateContent` | `Authorization: Bearer ...` | Outils Gemini natifs. |

## Règles de compatibilité

- Les formats natifs s'appliquent normalement à leur famille de modèles.
- Le format OpenAI compatible reste le choix recommandé, mais tous les paramètres ne sont pas garantis pour tous les modèles.
- Les embeddings utilisent `/v1/embeddings`.
- La disponibilité dépend de la console et des réponses API.

## Guide de choix

- En cas de doute, commencez par `/v1/chat/completions`.
- Pour les clients généraux, choisissez OpenAI Compatible Provider.
- Pour Claude Code ou les SDK Claude natifs, utilisez `/v1/messages`.
- Pour les outils Gemini natifs, utilisez `/v1beta/models/{model}:generateContent`.
