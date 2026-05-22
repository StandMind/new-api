ほとんどのクライアントでは Base URL のみ必要です。SDK が個別パスを追加できるよう、/v1 を優先してください。

| 用途 | 推奨値 | 備考 |
| --- | --- | --- |
| OpenAI SDK / 一般クライアント | https://aivrae.com/v1 | 推奨。 |
| 完全な chat endpoint | https://aivrae.com/v1/chat/completions | クライアントが要求する場合のみ。 |
| モデル一覧 | https://aivrae.com/v1/models | キーで見えるモデルを確認。 |
| Claude ネイティブ | https://aivrae.com/v1/messages | x-api-key と anthropic-version が必要。 |
| Gemini ネイティブ | https://aivrae.com/v1beta/models/{model}:generateContent | {model} を置換します。 |

```
推奨 Base URL
https://aivrae.com/v1

よく使う完全な endpoints
POST https://aivrae.com/v1/chat/completions
POST https://aivrae.com/v1/responses
POST https://aivrae.com/v1/completions
POST https://aivrae.com/v1/embeddings
GET  https://aivrae.com/v1/models

ネイティブテキスト endpoints
POST https://aivrae.com/v1/messages
POST https://aivrae.com/v1beta/models/{model}:generateContent
```
