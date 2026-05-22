Les applications doivent gérer explicitement les codes courants et journaliser les erreurs côté backend.

| Statut | Signification | Traitement conseillé |
| --- | --- | --- |
| 400 | Corps invalide, paramètre non pris en charge ou JSON mal formé | Vérifiez le corps, l'endpoint et les paramètres. |
| 401 | Clé manquante, erronée, expirée ou désactivée | Recopiez la clé et vérifiez l'en-tête. |
| 403 | Pas d'autorisation pour modèle, groupe ou fonction | Vérifiez permissions, plan et groupe. |
| 404 | Chemin ou modèle introuvable | Vérifiez Base URL, chemin natif et modèle. |
| 413 | Corps ou contexte trop grand | Réduisez messages, contents ou entrée. |
| 429 | Limite de fréquence, concurrence, quota ou solde | Réduisez la concurrence et vérifiez le solde. |
| 500 / 502 / 503 | Erreur temporaire Aivrae ou modèle | Réessayez plus tard ; contactez le support si persistant. |
