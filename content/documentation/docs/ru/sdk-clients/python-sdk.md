Python-настройка требует только api_key и base_url. Streaming нужно читать по частям.

## Базовый чат

```
from openai import OpenAI

client = OpenAI(
    api_key="YOUR_AIVRAE_API_KEY",
    base_url="https://aivrae.com/v1",
)

response = client.chat.completions.create(
    model="MODEL_NAME",
    messages=[{"role": "user", "content": "Привет, Aivrae"}],
)
```

## Streaming

```
from openai import OpenAI

client = OpenAI(
    api_key="YOUR_AIVRAE_API_KEY",
    base_url="https://aivrae.com/v1",
)

stream = client.chat.completions.create(
    model="MODEL_NAME",
    messages=[{"role": "user", "content": "Напиши хайку."}],
    stream=True,
)

for chunk in stream:
    delta = chunk.choices[0].delta
    if delta.content:
        print(delta.content, end="", flush=True)
```
