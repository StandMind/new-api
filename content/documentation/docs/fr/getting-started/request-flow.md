Les API texte Aivrae utilisent HTTPS et JSON. Ajoutez l'authentification, le modèle et le contenu d'entrée.

| Élément | Valeur | Notes |
| --- | --- | --- |
| Authentification | Authorization: Bearer YOUR_AIVRAE_API_KEY | Pour les endpoints OpenAI compatibles et Gemini natif. |
| Auth Claude | x-api-key: YOUR_AIVRAE_API_KEY | Pour le format Claude natif. |
| Type de contenu | Content-Type: application/json | Les endpoints texte utilisent JSON. |
| Nom de modèle | Copié depuis la console | Respectez la casse et les suffixes. |
| Timeout | 60 s ou plus | Le streaming garde la connexion ouverte. |
| Journaux | Temps, modèle, statut, erreur | Utile pour la facturation et le diagnostic. |
