非ストリーミングの Chat Completions は、生成が完了してから完全な JSON レスポンスを一度に返します。短い文章生成、分類、要約、構造化抽出、リアルタイム表示が不要なバックグラウンド処理に向いています。

Endpoint: `POST /v1/chat/completions`

> [!NOTE]
> リクエスト形式は OpenAI Chat Completions 互換です。利用できるモデル、対応パラメータ、課金はコンソールで有効化されたモデルと上流チャネルに依存します。

## ヘッダー

| パラメータ | 型 | 必須 | 説明 |
| --- | --- | --- | --- |
| Authorization | string | はい | `Bearer YOUR_API_KEY` を指定します。 |
| Content-Type | string | はい | `application/json` を指定します。 |

## リクエスト本文

| パラメータ | 型 | 必須 | 説明 |
| --- | --- | --- | --- |
| model | string | はい | 呼び出すモデル名。例: `gpt-5.4-mini`。モデル一覧またはコンソールで利用可能な名前を使います。 |
| messages | array | はい | 会話メッセージの配列。順番どおりにコンテキストになります。 |
| messages[].role | string | はい | メッセージの役割。主に `system`、`user`、`assistant`、`tool`。 |
| messages[].content | string/array | はい | メッセージ内容。テキストは string、対応モデルではマルチモーダル配列も使用できます。 |
| temperature | number | いいえ | サンプリングのランダム性。通常 `0` から `2`。 |
| top_p | number | いいえ | nucleus sampling。`temperature` と同時に大きく変更しないことを推奨します。 |
| max_tokens | integer | いいえ | 最大生成 token 数。例では `4096`。実際の上限はモデルと上流制限に依存します。 |
| stream | boolean | いいえ | 非ストリーミングでは省略するか `false` を指定します。 |
| stop | string/array | いいえ | 指定した文字列に到達したら生成を停止します。 |
| tools | array | いいえ | 関数またはツール定義。対応状況はモデルに依存します。 |
| tool_choice | string/object | いいえ | `auto`、`none`、特定ツールなどのツール選択を制御します。 |
| response_format | object | いいえ | JSON オブジェクトや JSON Schema など、出力形式を指定します。 |
| presence_penalty | number | いいえ | 同じ話題の繰り返しを抑制します。通常 `-2` から `2`。 |
| frequency_penalty | number | いいえ | 同じ表現の繰り返しを抑制します。通常 `-2` から `2`。 |
| user | string | いいえ | 監査やリスク管理用のエンドユーザー識別子。 |

## リクエスト例

```bash
curl https://aivrae.com/v1/chat/completions \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-5.4-mini",
    "messages": [
      { "role": "system", "content": "You are a concise assistant." },
      { "role": "user", "content": "Write a short welcome message." }
    ],
    "temperature": 0.7,
    "max_tokens": 4096
  }'
```

## レスポンス項目

| フィールド | 説明 |
| --- | --- |
| id | レスポンス ID。 |
| object | オブジェクト種別。通常 `chat.completion`。 |
| created | 作成時刻の Unix タイムスタンプ。 |
| model | 実際に応答したモデル名。 |
| choices[].message.role | 返却メッセージの役割。通常 `assistant`。 |
| choices[].message.content | 生成された本文。 |
| choices[].finish_reason | 終了理由。`stop`、`length`、`tool_calls` など。`length` は出力上限に到達したことを示します。 |
| usage.prompt_tokens | 入力 token 数。 |
| usage.completion_tokens | 出力 token 数。 |
| usage.total_tokens | 合計 token 数。課金確認に利用できます。 |

## 公式ドキュメント

- [OpenAI Chat Completions API](https://platform.openai.com/docs/api-reference/chat/create)
- [OpenAI Text generation guide](https://platform.openai.com/docs/guides/text-generation)
