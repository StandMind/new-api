Aivrae のテキスト API は HTTPS JSON リクエストです。認証、モデル名、入力内容を含めます。

| 項目 | 値 | 備考 |
| --- | --- | --- |
| 認証 | Authorization: Bearer YOUR_AIVRAE_API_KEY | OpenAI 互換と Gemini ネイティブで使用。 |
| Claude 認証 | x-api-key: YOUR_AIVRAE_API_KEY | Claude ネイティブ形式で使用。 |
| Content-Type | Content-Type: application/json | テキスト endpoint は JSON。 |
| モデル名 | コンソールからコピー | 大文字小文字と接尾辞が重要。 |
| タイムアウト | 60 秒以上推奨 | ストリーミングでは接続を維持します。 |
| ログ | 時刻、モデル、ステータス、エラー | 課金確認と調査に重要。 |
