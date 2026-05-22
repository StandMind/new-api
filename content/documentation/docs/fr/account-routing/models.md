Les noms de modèles doivent correspondre exactement à la console. Les modèles visibles peuvent varier selon le compte, le groupe ou le plan ; certains modèles ne prennent en charge que certains endpoints ou paramètres.

**Endpoint:** `GET /v1/models`

| Champ / concept | Description |
| --- | --- |
| id | Valeur à passer dans le champ model. |
| owned_by / provider | Identifie le fournisseur ou la route si présent. |
| Longueur de contexte | Suivez la console ; un contexte trop long peut renvoyer 400, 413 ou une erreur de service. |
| Compatibilité endpoint | Chat via chat completions, embeddings via embeddings, Claude via /v1/messages, Gemini via generateContent. |

```
curl https://aivrae.com/v1/models \
  -H "Authorization: Bearer YOUR_AIVRAE_API_KEY"
```
