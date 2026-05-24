L'API native Claude utilise le protocole Anthropic Messages. Elle est utile si votre client utilise deja les SDK Claude ou si vous avez besoin de champs natifs comme `system`, `tools`, `thinking` et `stream`. Pour une compatibilite large, preferez Chat Completions compatible OpenAI.

Endpoint: `POST /v1/messages`

> [!NOTE]
> Les requetes Claude natives utilisent l'en-tete `x-api-key` au lieu de `Authorization: Bearer ...`. Le testeur configure l'en-tete d'authentification adapte a cet endpoint.

## En-tetes

| Parametre | Type | Requis | Description |
| --- | --- | --- | --- |
| x-api-key | string | Oui | Votre API Key. |
| anthropic-version | string | Oui | Version de l'API Anthropic. L'exemple utilise `2023-06-01`. |
| Content-Type | string | Oui | Doit etre `application/json`. |

## Corps de requete

| Parametre | Type | Requis | Description |
| --- | --- | --- | --- |
| model | string | Oui | Nom du modele Claude active dans la console. |
| max_tokens | integer | Oui | Nombre maximal de tokens de sortie. L'exemple utilise `4096`. |
| messages | array | Oui | Messages de conversation. |
| messages[].role | string | Oui | Role du message, souvent `user` ou `assistant`. |
| messages[].content | string/array | Oui | Contenu du message: string pour texte ou blocs pour multimodal. |
| system | string/array | Non | Prompt systeme guidant l'assistant. |
| temperature | number | Non | Controle l'aleatoire. |
| top_p | number | Non | Echantillonnage nucleus. |
| top_k | integer | Non | Limite les tokens candidats. |
| stop_sequences | array | Non | Arrete la generation lorsqu'une sequence apparait. |
| stream | boolean | Non | `true` pour le streaming natif Claude. |
| tools | array | Non | Definitions d'outils. |
| tool_choice | object | Non | Controle la selection d'outil. |
| metadata | object | Non | Metadonnees utilisateur ou metier. |
| thinking | object | Non | Configuration de pensee etendue, seulement pour les modeles compatibles. |

## Exemple

```bash
curl https://aivrae.com/v1/messages \
  -H "x-api-key: YOUR_API_KEY" \
  -H "anthropic-version: 2023-06-01" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "claude-3-5-sonnet-20241022",
    "max_tokens": 4096,
    "messages": [
      { "role": "user", "content": "Write a concise project summary." }
    ]
  }'
```

## Champs de reponse

| Champ | Description |
| --- | --- |
| id | Identifiant de reponse Claude. |
| type | Type d'objet, souvent `message`. |
| role | Role retourne, souvent `assistant`. |
| content | Tableau de blocs de contenu. Le texte est souvent dans `content[].text`. |
| model | Modele ayant produit la reponse. |
| stop_reason | Raison d'arret: `end_turn`, `max_tokens` ou `tool_use`. |
| usage.input_tokens | Tokens d'entree. |
| usage.output_tokens | Tokens de sortie. |

## Documentation officielle

- [Anthropic Messages API](https://docs.anthropic.com/en/api/messages)
- [Anthropic streaming Messages](https://docs.anthropic.com/en/api/messages-streaming)
