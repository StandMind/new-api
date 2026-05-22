В OpenAI-совместимом SDK переопределите baseURL. Храните API-ключи в переменных окружения, а не во frontend-коде.

```
import OpenAI from "openai";

const client = new OpenAI({
  apiKey: process.env.AIVRAE_API_KEY,
  baseURL: "https://aivrae.com/v1",
});

const response = await client.chat.completions.create({
  model: "MODEL_NAME",
  messages: [{ role: "user", content: "Привет, Aivrae" }],
});
```
