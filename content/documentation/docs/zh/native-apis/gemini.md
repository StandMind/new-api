Gemini 原生接口使用 Google Gemini `generateContent` 风格的请求体，适合已经接入 Gemini SDK 或需要使用 Gemini 原生 `contents`、`generationConfig`、`safetySettings` 等能力的场景。只需要通用兼容性时，建议优先使用 OpenAI 兼容的 Chat Completions。

Endpoint: `POST /v1beta/models/{model}:generateContent`

> [!NOTE]
> `{model}` 放在 URL 路径中，例如 `gemini-2.5-flash`。具体可用模型和价格以控制台配置为准。

## 请求头

| 参数 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| Authorization | string | 是 | 使用 `Bearer YOUR_API_KEY`。 |
| Content-Type | string | 是 | 固定为 `application/json`。 |

## 路径参数

| 参数 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| model | string | 是 | Gemini 模型名称，例如 `gemini-2.5-flash`。调试器会把模型输入框同步到 URL 中的 `{model}`。 |

## 请求体参数

| 参数 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| contents | array | 是 | 输入内容数组。多轮对话按顺序追加多条 content。 |
| contents[].role | string | 否 | 角色，常用值为 `user`、`model`。单轮文本请求可只传 `user`。 |
| contents[].parts | array | 是 | 内容片段数组。文本、图片、文件等都通过 parts 表达。 |
| contents[].parts[].text | string | 否 | 文本输入内容。 |
| systemInstruction | object | 否 | 系统级指令，用于设定模型行为。 |
| generationConfig | object | 否 | 生成配置。 |
| generationConfig.temperature | number | 否 | 控制随机性。 |
| generationConfig.topP | number | 否 | 核采样参数。 |
| generationConfig.topK | integer | 否 | 从概率最高的 K 个 token 中采样。 |
| generationConfig.maxOutputTokens | integer | 否 | 最大输出 token 数。示例使用 `4096`。 |
| generationConfig.stopSequences | array | 否 | 遇到指定序列时停止生成。 |
| safetySettings | array | 否 | 安全策略设置，支持情况取决于上游。 |
| tools | array | 否 | Gemini 原生工具定义，例如 function calling。 |
| toolConfig | object | 否 | 工具调用配置。 |

## 示例请求

```bash
curl https://aivrae.com/v1beta/models/gemini-2.5-flash:generateContent \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "contents": [
      {
        "role": "user",
        "parts": [
          { "text": "Write a concise project summary." }
        ]
      }
    ],
    "generationConfig": {
      "temperature": 0.7,
      "maxOutputTokens": 4096
    }
  }'
```

## 响应字段

| 字段 | 说明 |
| --- | --- |
| candidates | 候选回复数组。通常读取第一个候选。 |
| candidates[].content.parts[].text | 生成的文本内容。 |
| candidates[].finishReason | 结束原因，例如 `STOP`、`MAX_TOKENS`、`SAFETY`。 |
| candidates[].safetyRatings | 安全评级信息。 |
| usageMetadata.promptTokenCount | 输入 token 数。 |
| usageMetadata.candidatesTokenCount | 输出 token 数。 |
| usageMetadata.totalTokenCount | 总 token 数。 |

## 官方文档

- [Gemini generateContent API](https://ai.google.dev/api/generate-content)
- [Gemini function calling](https://ai.google.dev/gemini-api/docs/function-calling)
