Gemini ネイティブ API は Google Gemini の `generateContent` 形式を使用します。Gemini SDK を利用している場合や、`contents`、`generationConfig`、`safetySettings` などのネイティブ項目が必要な場合に向いています。広いクライアント互換性が必要な場合は OpenAI 互換 Chat Completions を推奨します。

Endpoint: `POST /v1beta/models/{model}:generateContent`

> [!NOTE]
> `{model}` は URL パスの一部です。例: `gemini-2.5-flash`。利用可能なモデルと価格はコンソール設定に依存します。

## ヘッダー

| パラメータ | 型 | 必須 | 説明 |
| --- | --- | --- | --- |
| Authorization | string | はい | `Bearer YOUR_API_KEY` を指定します。 |
| Content-Type | string | はい | `application/json` を指定します。 |

## パスパラメータ

| パラメータ | 型 | 必須 | 説明 |
| --- | --- | --- | --- |
| model | string | はい | Gemini モデル名。例: `gemini-2.5-flash`。テスターのモデル欄は URL の `{model}` と同期します。 |

## リクエスト本文

| パラメータ | 型 | 必須 | 説明 |
| --- | --- | --- | --- |
| contents | array | はい | 入力コンテンツ配列。複数ターンでは順番に追加します。 |
| contents[].role | string | いいえ | 役割。通常 `user` または `model`。 |
| contents[].parts | array | はい | コンテンツ片。テキスト、画像、ファイルを parts で表します。 |
| contents[].parts[].text | string | いいえ | テキスト入力。 |
| systemInstruction | object | いいえ | モデルの振る舞いを指示するシステム指示。 |
| generationConfig | object | いいえ | 生成設定。 |
| generationConfig.temperature | number | いいえ | ランダム性を制御します。 |
| generationConfig.topP | number | いいえ | nucleus sampling。 |
| generationConfig.topK | integer | いいえ | 上位 K 個の候補 token からサンプリングします。 |
| generationConfig.maxOutputTokens | integer | いいえ | 最大出力 token 数。例では `4096`。 |
| generationConfig.stopSequences | array | いいえ | 指定シーケンスで生成を停止します。 |
| safetySettings | array | いいえ | 上流対応に依存する安全設定。 |
| tools | array | いいえ | function calling などの Gemini ネイティブツール定義。 |
| toolConfig | object | いいえ | ツール呼び出し設定。 |

## リクエスト例

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

## レスポンス項目

| フィールド | 説明 |
| --- | --- |
| candidates | 候補応答配列。多くのクライアントは最初の候補を読みます。 |
| candidates[].content.parts[].text | 生成されたテキスト。 |
| candidates[].finishReason | 終了理由。`STOP`、`MAX_TOKENS`、`SAFETY` など。 |
| candidates[].safetyRatings | 安全評価の詳細。 |
| usageMetadata.promptTokenCount | 入力 token 数。 |
| usageMetadata.candidatesTokenCount | 出力 token 数。 |
| usageMetadata.totalTokenCount | 合計 token 数。 |

## 公式ドキュメント

- [Gemini generateContent API](https://ai.google.dev/api/generate-content)
- [Gemini function calling](https://ai.google.dev/gemini-api/docs/function-calling)
