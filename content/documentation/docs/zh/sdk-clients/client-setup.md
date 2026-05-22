大多数第三方客户端都按 OpenAI Compatible Provider 接入。字段名称略有不同，但核心都是 API Key、Base URL、Model。

| 客户端 | Provider/类型 | Base URL | 备注 |
| --- | --- | --- | --- |
| Cherry Studio | OpenAI Compatible | https://aivrae.com/v1 | 模型名称从控制台复制。 |
| ChatBox | OpenAI API Compatible | https://aivrae.com/v1 | 推荐先做短文本测试。 |
| Cursor / Cline | OpenAI Compatible / Custom Provider | https://aivrae.com/v1 | 适合代码问答和自动化任务。 |
| Dify | OpenAI-API-compatible | https://aivrae.com/v1 | 按模型类型分别添加 Chat / Embedding。 |
| LobeChat / NextChat | 自定义 OpenAI 接口 | https://aivrae.com/v1 | 如果要求接口地址，填完整 chat completions。 |
| Aider / Claude Code | OpenAI compatible 或对应原生协议 | https://aivrae.com/v1 | Claude 原生模式使用 /v1/messages，Gemini 原生工具链使用 /v1beta/models/{model}:generateContent。 |
