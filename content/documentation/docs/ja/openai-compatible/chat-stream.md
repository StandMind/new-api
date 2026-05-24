ストリーミングの Chat Completions は、モデル生成中に Server-Sent Events を逐次返します。チャット UI、リアルタイムのターミナル出力、長い回答、初回 token の遅延を短くしたい場面に向いています。

Endpoint: `POST /v1/chat/completions`

> [!TIP]
> このページ下部の API テスターはストリーミング応答を読み取れます。リクエスト本文に `"stream": true` を設定すると、到着したチャンクがレスポンス欄に順次表示されます。

## リクエスト本文

| パラメータ | 型 | 必須 | 説明 |
| --- | --- | --- | --- |
| model | string | はい | 呼び出すモデル名。例: `gpt-5.4-mini`。 |
| messages | array | はい | 会話メッセージ。非ストリーミングと同じ構造です。 |
| stream | boolean | はい | ストリーミングを有効にするため `true` が必要です。 |
| stream_options | object | いいえ | ストリーミング追加オプション。 |
| stream_options.include_usage | boolean | いいえ | `true` の場合、対応上流では終了付近に usage が返ることがあります。 |
| temperature | number | いいえ | ランダム性を制御します。 |
| top_p | number | いいえ | nucleus sampling。 |
| max_tokens | integer | いいえ | 最大生成 token 数。例では `4096`。 |
| tools | array | いいえ | ツール定義。ストリーミング時のツール呼び出しは `delta.tool_calls` で返ります。 |
| tool_choice | string/object | いいえ | ツール選択を制御します。 |
| response_format | object | いいえ | 出力形式を指定します。JSON 出力ではプロンプトでも JSON を要求してください。 |

## SSE 形式

| フィールド | 説明 |
| --- | --- |
| data | SSE のデータ行。通常 JSON で、最後に `[DONE]` が来ることがあります。 |
| choices[].delta.content | このチャンクで追加されたテキスト。現在の回答に追記します。 |
| choices[].delta.role | ストリーム開始時に返ることがある役割情報。 |
| choices[].delta.tool_calls | ツール呼び出しの増分片。`index` ごとに結合します。 |
| choices[].finish_reason | 終了理由。空でなければその choice は終了です。 |
| usage | `stream_options.include_usage=true` かつ上流対応時に末尾付近で返ることがあります。 |

## リクエスト例

```bash
curl https://aivrae.com/v1/chat/completions \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-5.4-mini",
    "stream": true,
    "stream_options": { "include_usage": true },
    "messages": [
      { "role": "user", "content": "Give me three product name ideas." }
    ],
    "max_tokens": 4096
  }'
```

## ストリーム例

```text
data: {"choices":[{"delta":{"role":"assistant"},"index":0}]}

data: {"choices":[{"delta":{"content":"最初"},"index":0}]}

data: {"choices":[{"delta":{"content":"の案"},"index":0}]}

data: [DONE]
```

## クライアント側の注意

- 各 `data:` 行を読み、`delta.content` を現在の回答へ追記します。
- `data: [DONE]` を受け取ったら読み取りを終了します。
- ネットワーク切断後に同じ生成リクエストを無制限に再試行しないでください。重複生成や重複課金の原因になります。
- `finish_reason` が `length` の場合、`max_tokens` を増やすか入力を短くしてください。

## 公式ドキュメント

- [OpenAI Chat Completions API](https://platform.openai.com/docs/api-reference/chat/create)
- [OpenAI Streaming guide](https://platform.openai.com/docs/guides/streaming-responses)
