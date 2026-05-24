L'API native Gemini utilise le format Google Gemini `generateContent`. Elle est utile si votre client utilise deja les SDK Gemini ou si vous avez besoin de champs natifs comme `contents`, `generationConfig` et `safetySettings`. Pour une compatibilite large, preferez Chat Completions compatible OpenAI.

Endpoint: `POST /v1beta/models/{model}:generateContent`

> [!NOTE]
> `{model}` fait partie du chemin, par exemple `gemini-2.5-flash`. Les modeles et tarifs disponibles dependent de la configuration de la console.

## En-tetes

| Parametre | Type | Requis | Description |
| --- | --- | --- | --- |
| Authorization | string | Oui | Utiliser `Bearer YOUR_API_KEY`. |
| Content-Type | string | Oui | Doit etre `application/json`. |

## Parametres de chemin

| Parametre | Type | Requis | Description |
| --- | --- | --- | --- |
| model | string | Oui | Nom du modele Gemini, par exemple `gemini-2.5-flash`. Le testeur synchronise le champ modele avec `{model}` dans l'URL. |

## Corps de requete

| Parametre | Type | Requis | Description |
| --- | --- | --- | --- |
| contents | array | Oui | Tableau de contenu d'entree. Ajoutez plusieurs elements pour une conversation multi-tour. |
| contents[].role | string | Non | Role, souvent `user` ou `model`. |
| contents[].parts | array | Oui | Parties du contenu: texte, images ou fichiers. |
| contents[].parts[].text | string | Non | Texte d'entree. |
| systemInstruction | object | Non | Instruction systeme guidant le comportement du modele. |
| generationConfig | object | Non | Parametres de generation. |
| generationConfig.temperature | number | Non | Controle l'aleatoire. |
| generationConfig.topP | number | Non | Echantillonnage nucleus. |
| generationConfig.topK | integer | Non | Echantillonne parmi les K tokens les plus probables. |
| generationConfig.maxOutputTokens | integer | Non | Nombre maximal de tokens de sortie. L'exemple utilise `4096`. |
| generationConfig.stopSequences | array | Non | Arrete la generation lorsqu'une sequence apparait. |
| safetySettings | array | Non | Parametres de securite selon le support upstream. |
| tools | array | Non | Definitions d'outils natives Gemini, comme function calling. |
| toolConfig | object | Non | Configuration d'appel d'outils. |

## Exemple

```bash
curl https://aivrae.com/v1beta/models/gemini-2.5-flash:generateContent \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "contents": [
      {
        "role": "user",
        "parts": [
          { "text": "Write a concise project summary." }
        ]
      }
    ],
    "generationConfig": {
      "temperature": 0.7,
      "maxOutputTokens": 4096
    }
  }'
```

## Champs de reponse

| Champ | Description |
| --- | --- |
| candidates | Tableau de reponses candidates. La plupart des clients lisent la premiere. |
| candidates[].content.parts[].text | Texte genere. |
| candidates[].finishReason | Raison d'arret: `STOP`, `MAX_TOKENS` ou `SAFETY`. |
| candidates[].safetyRatings | Details d'evaluation de securite. |
| usageMetadata.promptTokenCount | Tokens d'entree. |
| usageMetadata.candidatesTokenCount | Tokens de sortie. |
| usageMetadata.totalTokenCount | Total des tokens. |

## Documentation officielle

- [Gemini generateContent API](https://ai.google.dev/api/generate-content)
- [Gemini function calling](https://ai.google.dev/gemini-api/docs/function-calling)
