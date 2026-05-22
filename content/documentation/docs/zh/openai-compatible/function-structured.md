函数调用和结构化输出仍属于文本接口范围，但是否可用取决于模型能力和路由兼容性。生产环境应对不支持的模型做降级处理。

**Endpoint:** `POST /v1/chat/completions`

## 函数调用示例

```
curl https://aivrae.com/v1/chat/completions \
  -H "Authorization: Bearer YOUR_AIVRAE_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "MODEL_NAME",
    "messages": [
      { "role": "user", "content": "What is the weather in Shanghai?" }
    ],
    "tools": [
      {
        "type": "function",
        "function": {
          "name": "get_weather",
          "description": "Get current weather",
          "parameters": {
            "type": "object",
            "properties": {
              "city": { "type": "string" }
            },
            "required": ["city"]
          }
        }
      }
    ],
    "tool_choice": "auto"
  }'
```

## JSON 输出示例

```
curl https://aivrae.com/v1/chat/completions \
  -H "Authorization: Bearer YOUR_AIVRAE_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "MODEL_NAME",
    "response_format": { "type": "json_object" },
    "messages": [
      { "role": "system", "content": "Return valid JSON only." },
      { "role": "user", "content": "Generate a JSON object with name and slogan." }
    ]
  }'
```
