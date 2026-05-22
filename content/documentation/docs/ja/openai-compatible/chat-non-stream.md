非ストリーミングは生成完了後に結果をまとめて返します。短文、バックグラウンド処理、分類、要約に適しています。

**Endpoint:** `POST /v1/chat/completions`

| パラメータ | 必須 | 説明 |
| --- | --- | --- |
| model | はい | コンソールのモデル名。 |
| messages | はい | system、user、assistant などの会話配列。 |
| temperature | いいえ | 0〜2。一般的には 0.2〜0.8。 |
| max_tokens | いいえ | 出力長を制限。 |
| stream | いいえ | 非ストリームでは省略または false。 |

```
curl https://aivrae.com/v1/chat/completions \
  -H "Authorization: Bearer YOUR_AIVRAE_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "MODEL_NAME",
    "messages": [
      { "role": "system", "content": "あなたは簡潔に答えるアシスタントです。" },
      { "role": "user", "content": "短い歓迎メッセージを書いてください。" }
    ],
    "temperature": 0.7,
    "max_tokens": 512
  }'
```

## よく使うレスポンスフィールド

| フィールド | 説明 |
| --- | --- |
| choices[0].message.content | モデルが返す主要テキスト。 |
| choices[0].finish_reason | stop、length、tool_calls などの終了理由。 |
| usage.prompt_tokens | 入力トークン数。 |
| usage.completion_tokens | 出力トークン数。 |
| usage.total_tokens | 合計トークン数。通常は課金参照に使います。 |
