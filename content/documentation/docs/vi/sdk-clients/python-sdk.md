Python chỉ cần api_key và base_url. Phản hồi streaming nên đọc theo từng chunk.

## Chat cơ bản

```
from openai import OpenAI

client = OpenAI(
    api_key="YOUR_AIVRAE_API_KEY",
    base_url="https://aivrae.com/v1",
)

response = client.chat.completions.create(
    model="MODEL_NAME",
    messages=[{"role": "user", "content": "Xin chào Aivrae"}],
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
    messages=[{"role": "user", "content": "Viết một bài haiku."}],
    stream=True,
)

for chunk in stream:
    delta = chunk.choices[0].delta
    if delta.content:
        print(delta.content, end="", flush=True)
```
