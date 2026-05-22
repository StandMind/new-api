Les modèles de raisonnement peuvent retourner reasoning_content, reasoning, thinking ou du contenu entouré de <think>...</think>. Les noms de champs varient selon le modèle et la route.

| Champ / forme | Source possible | Traitement |
| --- | --- | --- |
| delta.reasoning_content | Modèles de raisonnement comme DeepSeek | Afficher séparément ou replier. |
| reasoning / thinking | Gemini ou routes compatibles | Ne pas supposer que tous les modèles l'ont. |
| <think>...</think> | Certaines routes compatibles | Filtrer si vous n'affichez pas le raisonnement. |
| delta.content | Réponse finale | Texte principal destiné à l'utilisateur. |
