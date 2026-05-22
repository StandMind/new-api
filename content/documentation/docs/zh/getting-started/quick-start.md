从零完成一次可用的文本模型调用，建议按下面顺序操作。

- 注册并登录 Aivrae 控制台。
- 充值或确认账户有可用预付费额度。
- 进入令牌/API Key 页面，点击创建令牌。
- 如果控制台提供分组或渠道选择，先使用默认或推荐分组完成测试。
- 设置 Base URL 为 https://aivrae.com/v1，复制控制台中的模型名称发起请求。

```
{
  "base_url": "https://aivrae.com/v1",
  "api_key": "YOUR_AIVRAE_API_KEY",
  "model": "MODEL_NAME"
}
```
