非流式请求会在模型生成结束后一次性返回完整结果，适合短文本、后台任务、分类、摘要和不需要实时展示的业务。

**Endpoint:** `POST /v1/chat/completions`

| 参数 | 必填 | 说明 |
| --- | --- | --- |
| model | 是 | 控制台复制的模型名称。 |
| messages | 是 | 对话消息数组，常见 role 包括 system、user、assistant。 |
| temperature | 否 | 0 到 2 之间，越高越随机；多数场景使用 0.2 到 0.8。 |
| max_tokens | 否 | 限制输出长度，实际可用长度受模型上下文限制。 |
| stream | 否 | 非流式不传或传 false。 |

```
curl https://aivrae.com/v1/chat/completions \
  -H "Authorization: Bearer YOUR_AIVRAE_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "MODEL_NAME",
    "messages": [
      { "role": "system", "content": "You are a concise assistant." },
      { "role": "user", "content": "Write a short welcome message." }
    ],
    "temperature": 0.7,
    "max_tokens": 512
  }'
```

## 常见响应字段

| 字段 | 说明 |
| --- | --- |
| choices[0].message.content | 模型返回的主要文本。 |
| choices[0].finish_reason | stop、length、tool_calls 等结束原因。 |
| usage.prompt_tokens | 输入 token 数。 |
| usage.completion_tokens | 输出 token 数。 |
| usage.total_tokens | 总 token 数，通常用于计费参考。 |
