多くのクライアントは OpenAI Compatible Provider として接続します。名称は違っても、必要なのは API Key、Base URL、Model です。

| クライアント | 種類 | Base URL | 備考 |
| --- | --- | --- | --- |
| Cherry Studio | OpenAI Compatible | https://aivrae.com/v1 | モデル名はコンソールから。 |
| ChatBox | OpenAI API Compatible | https://aivrae.com/v1 | まず短文テスト。 |
| Cursor / Cline | OpenAI Compatible / Custom Provider | https://aivrae.com/v1 | コードと agent に便利。 |
| Dify | OpenAI-API-compatible | https://aivrae.com/v1 | Chat と Embedding を別々に追加。 |
| LobeChat / NextChat | Custom OpenAI endpoint | https://aivrae.com/v1 | 必要時のみ完全 endpoint。 |
| Aider / Claude Code | OpenAI compatible または native | https://aivrae.com/v1 | Claude は /v1/messages、Gemini は /v1beta/models/{model}:generateContent。 |
