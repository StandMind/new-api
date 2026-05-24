Chat Completions sans streaming renvoie une reponse JSON complete une fois la generation terminee. Ce mode convient aux textes courts, classifications, resumes, extractions structurees et traitements en arriere-plan qui ne nécessitent pas d'affichage en temps reel.

Endpoint: `POST /v1/chat/completions`

> [!NOTE]
> Le format suit le style OpenAI Chat Completions. Les modeles disponibles, les parametres pris en charge et la facturation dependent des modeles et canaux upstream actives dans la console.

## En-tetes

| Parametre | Type | Requis | Description |
| --- | --- | --- | --- |
| Authorization | string | Oui | Utiliser `Bearer YOUR_API_KEY`. |
| Content-Type | string | Oui | Doit etre `application/json`. |

## Corps de requete

| Parametre | Type | Requis | Description |
| --- | --- | --- | --- |
| model | string | Oui | Nom du modele, par exemple `gpt-5.4-mini`. Utilisez un modele disponible dans la console. |
| messages | array | Oui | Messages de conversation ordonnes. |
| messages[].role | string | Oui | Role du message: `system`, `user`, `assistant` ou `tool`. |
| messages[].content | string/array | Oui | Contenu du message. Texte simple en string, contenu multimodal en array si le modele le prend en charge. |
| temperature | number | Non | Aleatoire d'echantillonnage, generalement entre `0` et `2`. |
| top_p | number | Non | Echantillonnage nucleus. Evitez de modifier fortement `temperature` et `top_p` en meme temps. |
| max_tokens | integer | Non | Nombre maximal de tokens generes. L'exemple utilise `4096`; la limite reelle depend du modele et de l'upstream. |
| stream | boolean | Non | Omettre ou definir `false` pour une reponse non streaming. |
| stop | string/array | Non | Arrete la generation lorsqu'une sequence apparait. |
| tools | array | Non | Definitions de fonctions ou d'outils, selon le modele. |
| tool_choice | string/object | Non | Controle le choix d'outil: `auto`, `none` ou outil specifique. |
| response_format | object | Non | Demande un format de sortie, par exemple JSON ou JSON Schema. |
| presence_penalty | number | Non | Penalise les sujets repetes, souvent entre `-2` et `2`. |
| frequency_penalty | number | Non | Penalise les formulations repetees, souvent entre `-2` et `2`. |
| user | string | Non | Identifiant utilisateur final pour audit et controle de risque. |

## Exemple

```bash
curl https://aivrae.com/v1/chat/completions \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-5.4-mini",
    "messages": [
      { "role": "system", "content": "You are a concise assistant." },
      { "role": "user", "content": "Write a short welcome message." }
    ],
    "temperature": 0.7,
    "max_tokens": 4096
  }'
```

## Champs de reponse

| Champ | Description |
| --- | --- |
| id | Identifiant de la reponse. |
| object | Type d'objet, souvent `chat.completion`. |
| created | Horodatage Unix de creation. |
| model | Modele ayant produit la reponse. |
| choices[].message.role | Role retourne, souvent `assistant`. |
| choices[].message.content | Texte principal genere. |
| choices[].finish_reason | Raison d'arret: `stop`, `length`, `tool_calls`. `length` signifie que la limite de sortie est atteinte. |
| usage.prompt_tokens | Tokens d'entree. |
| usage.completion_tokens | Tokens de sortie. |
| usage.total_tokens | Total des tokens, utile pour verifier la facturation. |

## Documentation officielle

- [OpenAI Chat Completions API](https://platform.openai.com/docs/api-reference/chat/create)
- [OpenAI Text generation guide](https://platform.openai.com/docs/guides/text-generation)
