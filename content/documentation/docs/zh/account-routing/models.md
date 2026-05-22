模型名称必须与控制台完全一致。不同账号、分组或套餐可见的模型可能不同；部分模型只支持特定接口或参数。

**Endpoint:** `GET /v1/models`

| 字段/概念 | 说明 |
| --- | --- |
| id | 请求中的 model 字段应填写的值。 |
| owned_by / provider | 如果存在，用于标识提供方或路由类型。 |
| 上下文长度 | 以控制台说明为准；超出上下文会返回 400、413 或服务错误。 |
| 接口兼容性 | 聊天模型使用 chat completions，Embedding 模型使用 embeddings，Claude 原生模型使用 /v1/messages，Gemini 原生模型使用 generateContent。 |

```
curl https://aivrae.com/v1/models \
  -H "Authorization: Bearer YOUR_AIVRAE_API_KEY"
```
