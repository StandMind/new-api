Với SDK tương thích OpenAI, hãy ghi đè baseURL. Lưu API key trong biến môi trường, không đặt trong frontend.

```
import OpenAI from "openai";

const client = new OpenAI({
  apiKey: process.env.AIVRAE_API_KEY,
  baseURL: "https://aivrae.com/v1",
});

const response = await client.chat.completions.create({
  model: "MODEL_NAME",
  messages: [{ role: "user", content: "Xin chào Aivrae" }],
});
```
