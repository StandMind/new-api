正式接入前，建议先在本文档内置的在线调试框完成一次小额度测试，确认 Key、模型名称、Base URL 和余额均正确。

- 在控制台创建 API Key，并复制一个可用模型名称。
- 在下方在线调试框中填写 API Key，必要时把默认模型名改成控制台可用模型。
- 点击发送请求后，确认响应区能正常返回状态码和模型输出。
- 如果后续使用 ChatBox、Cherry Studio、Postman 或 curl，也可以直接复用文档中的 Base URL 和请求体。
- 如果客户端有 OpenAI Compatible、Custom OpenAI、OpenAI API Compatible Provider 等选项，优先选择该模式。
- 测试阶段不要对大模型做高频健康检查，这会消耗额度并可能触发限流。
