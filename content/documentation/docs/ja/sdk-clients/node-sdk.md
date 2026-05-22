OpenAI 互換 SDK では baseURL を上書きします。API キーは frontend ではなく環境変数に保存します。

```
import OpenAI from "openai";

const client = new OpenAI({
  apiKey: process.env.AIVRAE_API_KEY,
  baseURL: "https://aivrae.com/v1",
});

const response = await client.chat.completions.create({
  model: "MODEL_NAME",
  messages: [{ role: "user", content: "こんにちは Aivrae" }],
});
```
