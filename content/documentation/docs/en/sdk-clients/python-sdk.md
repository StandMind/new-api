Python setup only needs api_key and base_url. Streaming responses should be consumed chunk by chunk.

## Basic chat

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

## Streaming output

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
