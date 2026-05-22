La configuration Python ne demande que api_key et base_url. Les réponses streaming doivent être lues bloc par bloc.

## Chat de base

```
from openai import OpenAI

client = OpenAI(
    api_key="YOUR_AIVRAE_API_KEY",
    base_url="https://aivrae.com/v1",
)

response = client.chat.completions.create(
    model="MODEL_NAME",
    messages=[{"role": "user", "content": "Bonjour Aivrae"}],
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
    messages=[{"role": "user", "content": "Écris un haïku."}],
    stream=True,
)

for chunk in stream:
    delta = chunk.choices[0].delta
    if delta.content:
        print(delta.content, end="", flush=True)
```
