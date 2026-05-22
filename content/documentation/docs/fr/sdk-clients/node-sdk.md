Avec le SDK compatible OpenAI, remplacez baseURL. Stockez les clés API dans des variables d'environnement, pas dans le frontend.

```
import OpenAI from "openai";

const client = new OpenAI({
  apiKey: process.env.AIVRAE_API_KEY,
  baseURL: "https://aivrae.com/v1",
});

const response = await client.chat.completions.create({
  model: "MODEL_NAME",
  messages: [{ role: "user", content: "Bonjour Aivrae" }],
});
```
