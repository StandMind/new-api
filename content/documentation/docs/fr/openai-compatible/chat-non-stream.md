Les requêtes non streaming renvoient le résultat complet après génération. Elles conviennent aux textes courts, tâches backend, classifications et résumés.

**Endpoint:** `POST /v1/chat/completions`

| Paramètre | Obligatoire | Description |
| --- | --- | --- |
| model | Oui | Nom de modèle copié depuis la console. |
| messages | Oui | Tableau de messages avec roles system, user, assistant. |
| temperature | Non | De 0 à 2 ; valeurs courantes 0.2 à 0.8. |
| max_tokens | Non | Limite la longueur de sortie. |
| stream | Non | Omettre ou false pour le non streaming. |

```
curl https://aivrae.com/v1/chat/completions \
  -H "Authorization: Bearer YOUR_AIVRAE_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "MODEL_NAME",
    "messages": [
      { "role": "system", "content": "Tu es un assistant concis." },
      { "role": "user", "content": "Écris un court message de bienvenue." }
    ],
    "temperature": 0.7,
    "max_tokens": 512
  }'
```

## Champs de réponse courants

| Champ | Description |
| --- | --- |
| choices[0].message.content | Texte principal renvoyé par le modèle. |
| choices[0].finish_reason | stop, length, tool_calls ou raison similaire. |
| usage.prompt_tokens | Nombre de tokens en entrée. |
| usage.completion_tokens | Nombre de tokens en sortie. |
| usage.total_tokens | Total de tokens, souvent utilisé comme référence de facturation. |
