Chat Completions en streaming renvoie des Server-Sent Events pendant que le modele genere. Ce mode convient aux interfaces de chat, sorties terminal en temps reel, reponses longues et cas ou la latence du premier token compte.

Endpoint: `POST /v1/chat/completions`

> [!TIP]
> Le testeur API en bas de page lit maintenant les reponses streaming. Ajoutez `"stream": true` au corps de requete et les fragments apparaitront au fur et a mesure.

## Corps de requete

| Parametre | Type | Requis | Description |
| --- | --- | --- | --- |
| model | string | Oui | Nom du modele, par exemple `gpt-5.4-mini`. |
| messages | array | Oui | Messages ordonnes, meme structure que la version non streaming. |
| stream | boolean | Oui | Doit etre `true` pour activer le streaming. |
| stream_options | object | Non | Options supplementaires du streaming. |
| stream_options.include_usage | boolean | Non | Si `true`, les upstreams compatibles peuvent envoyer l'usage en fin de flux. |
| temperature | number | Non | Controle l'aleatoire. |
| top_p | number | Non | Echantillonnage nucleus. |
| max_tokens | integer | Non | Nombre maximal de tokens generes. L'exemple utilise `4096`. |
| tools | array | Non | Definitions d'outils. Les appels d'outils arrivent via `delta.tool_calls`. |
| tool_choice | string/object | Non | Controle le choix d'outil. |
| response_format | object | Non | Demande un format de sortie. Pour JSON, demandez aussi explicitement du JSON dans le prompt. |

## Format SSE

| Champ | Description |
| --- | --- |
| data | Ligne de donnees SSE, generalement JSON ou `[DONE]`. |
| choices[].delta.content | Nouveau texte de ce fragment, a ajouter a la reponse courante. |
| choices[].delta.role | Role pouvant apparaitre au debut du flux. |
| choices[].delta.tool_calls | Fragments incrementaux d'appels d'outils, a fusionner par `index`. |
| choices[].finish_reason | Raison d'arret. Une valeur non vide indique que ce choice est termine. |
| usage | Peut apparaitre a la fin si `stream_options.include_usage=true` et si l'upstream le prend en charge. |

## Exemple

```bash
curl https://aivrae.com/v1/chat/completions \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-5.4-mini",
    "stream": true,
    "stream_options": { "include_usage": true },
    "messages": [
      { "role": "user", "content": "Give me three product name ideas." }
    ],
    "max_tokens": 4096
  }'
```

## Exemple de flux

```text
data: {"choices":[{"delta":{"role":"assistant"},"index":0}]}

data: {"choices":[{"delta":{"content":"Premiere"},"index":0}]}

data: {"choices":[{"delta":{"content":" idee"},"index":0}]}

data: [DONE]
```

## Notes client

- Lisez chaque ligne `data:` et ajoutez `delta.content` a la reponse courante.
- Fermez le lecteur a la reception de `data: [DONE]`.
- Ne relancez pas indefiniment la meme generation apres une coupure reseau, afin d'eviter doublons et double facturation.
- Si `finish_reason` vaut `length`, augmentez `max_tokens` ou raccourcissez l'entree.

## Documentation officielle

- [OpenAI Chat Completions API](https://platform.openai.com/docs/api-reference/chat/create)
- [OpenAI Streaming guide](https://platform.openai.com/docs/guides/streaming-responses)
