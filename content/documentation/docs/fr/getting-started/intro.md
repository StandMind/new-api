## Présentation de la plateforme

Aivrae est une passerelle d'agrégation d'API IA pour les développeurs et équipes internationaux. Elle fournit un endpoint unifié, la gestion des clés API, l'accès aux modèles, la comptabilisation des crédits prépayés et les journaux de requêtes. Vous pouvez utiliser les formats compatibles OpenAI pour le chat texte, la génération, les embeddings, l'API Responses, ainsi que les formats natifs Claude et Gemini.

Cette documentation publique couvre les services texte. Les fonctions disponibles dépendent de la console, des réponses API et du support officiel Aivrae.

## Capacités clés

- Accès unifié : utilisez `https://aivrae.com/v1` comme Base URL compatible OpenAI.
- Compatibilité : le format OpenAI compatible est recommandé par défaut, avec des exemples Claude Messages et Gemini generateContent.
- Changement de modèle : utilisez les noms affichés dans la console.
- Suivi d'utilisation : consultez solde, historique, consommation et erreurs dans la console.
- Test intégré : les pages d'API contiennent des valeurs de requête prêtes à tester.

## Parcours recommandé

1. Connectez-vous à la console Aivrae.
2. Créez une clé API et copiez un nom de modèle disponible.
3. Lancez une petite requête depuis le testeur intégré.
4. Configurez votre Base URL sur `https://aivrae.com/v1`.
5. En production, journalisez l'ID de requête, le code d'état, le modèle et les erreurs.

## Comparaison d'utilisation

| Élément | Avec Aivrae | Intégrations séparées |
| --- | --- | --- |
| Endpoint | Une Base URL compatible OpenAI plus certains endpoints natifs texte | Plusieurs endpoints à maintenir |
| API key | Gestion centralisée dans une console | Gestion séparée par service |
| Compatibilité client | Compatible avec la plupart des clients OpenAI-compatible | Configuration propre à chaque service |
| Changement de modèle | Tester les modèles disponibles en modifiant le nom du modèle | Souvent besoin de changer endpoint et paramètres |
| Journaux d'utilisation | Centralisés dans la console | Répartis dans plusieurs tableaux de bord |
| Test documentaire | Tester l'endpoint courant directement sur la page | Nécessite souvent Postman ou du code personnalisé |

## Portée actuelle

| Capacité | État | Notes |
| --- | --- | --- |
| Chat texte | Pris en charge | Utilisez `/v1/chat/completions`. |
| Streaming | Pris en charge | Activez `stream: true`. |
| Function calling / JSON | Selon le modèle | Dépend de `tools` et `response_format`. |
| Responses API | Pris en charge | Utilisez `/v1/responses`. |
| Embeddings | Pris en charge | Recherche sémantique, RAG et similarité. |
| Format Claude natif | Selon le modèle | `/v1/messages`, généralement pour les modèles Claude. |
| Format Gemini natif | Selon le modèle | `/v1beta/models/{model}:generateContent`, généralement pour Gemini. |
