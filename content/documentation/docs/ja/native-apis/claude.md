Claude ネイティブ API は Anthropic Messages プロトコルを使用します。Claude SDK を利用している場合や、`system`、`tools`、`thinking`、`stream` などのネイティブ項目が必要な場合に向いています。広いクライアント互換性が必要な場合は OpenAI 互換 Chat Completions を推奨します。

Endpoint: `POST /v1/messages`

> [!NOTE]
> Claude ネイティブリクエストは `Authorization: Bearer ...` ではなく `x-api-key` ヘッダーを使います。テスターはこの endpoint に合わせて認証ヘッダーを設定します。

## ヘッダー

| パラメータ | 型 | 必須 | 説明 |
| --- | --- | --- | --- |
| x-api-key | string | はい | API Key。 |
| anthropic-version | string | はい | Anthropic API バージョン。例では `2023-06-01`。 |
| Content-Type | string | はい | `application/json` を指定します。 |

## リクエスト本文

| パラメータ | 型 | 必須 | 説明 |
| --- | --- | --- | --- |
| model | string | はい | Claude モデル名。コンソールで有効なモデルを使います。 |
| max_tokens | integer | はい | 最大出力 token 数。例では `4096`。 |
| messages | array | はい | 会話メッセージ。 |
| messages[].role | string | はい | メッセージの役割。通常 `user` または `assistant`。 |
| messages[].content | string/array | はい | メッセージ内容。テキストは string、マルチモーダルはコンテンツブロック配列。 |
| system | string/array | いいえ | アシスタントの振る舞いを指示するシステムプロンプト。 |
| temperature | number | いいえ | ランダム性を制御します。 |
| top_p | number | いいえ | nucleus sampling。 |
| top_k | integer | いいえ | 候補 token 数を制限します。 |
| stop_sequences | array | いいえ | 指定シーケンスで生成を停止します。 |
| stream | boolean | いいえ | `true` で Claude ネイティブストリーミングを使用します。 |
| tools | array | いいえ | ツール定義。 |
| tool_choice | object | いいえ | ツール選択を制御します。 |
| metadata | object | いいえ | 任意のユーザーまたは業務メタデータ。 |
| thinking | object | いいえ | 拡張思考設定。対応モデルのみ有効です。 |

## リクエスト例

```bash
curl https://aivrae.com/v1/messages \
  -H "x-api-key: YOUR_API_KEY" \
  -H "anthropic-version: 2023-06-01" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "claude-3-5-sonnet-20241022",
    "max_tokens": 4096,
    "messages": [
      { "role": "user", "content": "Write a concise project summary." }
    ]
  }'
```

## レスポンス項目

| フィールド | 説明 |
| --- | --- |
| id | Claude レスポンス ID。 |
| type | オブジェクト種別。通常 `message`。 |
| role | 返却ロール。通常 `assistant`。 |
| content | コンテンツブロック配列。テキストは通常 `content[].text`。 |
| model | 実際に応答したモデル。 |
| stop_reason | 終了理由。`end_turn`、`max_tokens`、`tool_use` など。 |
| usage.input_tokens | 入力 token 数。 |
| usage.output_tokens | 出力 token 数。 |

## 公式ドキュメント

- [Anthropic Messages API](https://docs.anthropic.com/en/api/messages)
- [Anthropic streaming Messages](https://docs.anthropic.com/en/api/messages-streaming)
