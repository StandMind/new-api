Aivrae は OpenAI 互換 Chat Completions をデフォルト形式として推奨します。一部のモデルファミリーは Claude Messages や Gemini generateContent のネイティブ形式にも対応します。

| 形式 | 対象モデル | Endpoint | 認証 | 推奨用途 |
| --- | --- | --- | --- | --- |
| OpenAI 互換 | コンソール上の多くのテキストモデル | `/v1/chat/completions` | `Authorization: Bearer ...` | デフォルト。 |
| Claude ネイティブ | Claude モデル | `/v1/messages` | `x-api-key: ...` | Claude Code、Claude SDK。 |
| Gemini ネイティブ | Gemini モデル | `/v1beta/models/{model}:generateContent` | `Authorization: Bearer ...` | Gemini ネイティブツール。 |

## 互換性ルール

- ネイティブ形式は通常、対応するモデルファミリー専用です。
- OpenAI 互換形式を推奨しますが、全モデルが全パラメータに対応するとは限りません。
- Embeddings は `/v1/embeddings` を使います。
- 利用可否はコンソール権限と API レスポンスで確認します。

## 選択ガイド

- 迷った場合は `/v1/chat/completions` から始めます。
- 一般的なクライアントでは OpenAI Compatible Provider を選びます。
- Claude Code または Claude ネイティブ SDK では `/v1/messages` を使います。
- Gemini ネイティブツールでは `/v1beta/models/{model}:generateContent` を使います。
