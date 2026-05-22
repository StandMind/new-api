Python SDK 只需要设置 api_key 和 base_url。流式响应需要逐块读取。

## 基础对话

```
from openai import OpenAI

client = OpenAI(
    api_key="YOUR_AIVRAE_API_KEY",
    base_url="https://aivrae.com/v1",
)

response = client.chat.completions.create(
    model="MODEL_NAME",
    messages=[{"role": "user", "content": "Hello Aivrae"}],
)
```

## 流式输出

```
from openai import OpenAI

client = OpenAI(
    api_key="YOUR_AIVRAE_API_KEY",
    base_url="https://aivrae.com/v1",
)

stream = client.chat.completions.create(
    model="MODEL_NAME",
    messages=[{"role": "user", "content": "Write a haiku."}],
    stream=True,
)

for chunk in stream:
    delta = chunk.choices[0].delta
    if delta.content:
        print(delta.content, end="", flush=True)
```
