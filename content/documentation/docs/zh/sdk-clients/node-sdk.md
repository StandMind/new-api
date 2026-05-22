OpenAI 官方兼容 SDK 只需要覆盖 baseURL。API Key 建议放在环境变量中，不要写入前端代码。

```
import OpenAI from "openai";

const client = new OpenAI({
  apiKey: process.env.AIVRAE_API_KEY,
  baseURL: "https://aivrae.com/v1",
});

const response = await client.chat.completions.create({
  model: "MODEL_NAME",
  messages: [{ role: "user", content: "Hello Aivrae" }],
});
```
