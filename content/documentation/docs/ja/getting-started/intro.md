## プラットフォーム概要

Aivrae は、グローバルな開発者とチーム向けの AI API 集約ゲートウェイです。統一 API エンドポイント、API キー管理、モデルアクセス、プリペイド利用量の集計、リクエストログを提供します。OpenAI 互換形式、Responses API、embeddings、Claude ネイティブ形式、Gemini ネイティブ形式を利用できます。

この公開ドキュメントはテキストサービスを対象にしています。利用可能な機能は、コンソール表示、API レスポンス、Aivrae 公式サポートの案内に従います。

## 主な機能

- 統一アクセス：OpenAI 互換の Base URL として `https://aivrae.com/v1` を使用します。
- プロトコル互換：OpenAI 互換形式を推奨し、Claude Messages と Gemini generateContent の例も用意しています。
- モデル切替：コンソールに表示されるモデル名を使用します。
- 利用状況：残高、履歴、消費、エラーをコンソールで確認できます。
- 組み込みテスト：API ページに初期リクエスト値があり、その場でテストできます。

## 推奨手順

1. Aivrae コンソールにログインします。
2. API キーを作成し、利用可能なモデル名をコピーします。
3. 組み込みテスターで小さなリクエストを送信します。
4. アプリまたはクライアントの Base URL を `https://aivrae.com/v1` に設定します。
5. 本番では request ID、ステータス、モデル名、エラーを記録します。

## 利用方法の比較

| 項目 | Aivrae を使う場合 | 個別に連携する場合 |
| --- | --- | --- |
| Endpoint | 1つの OpenAI 互換 Base URL と一部のネイティブテキスト endpoints | 複数の endpoints を管理 |
| API キー | 1つのコンソールで管理 | サービスごとに管理 |
| クライアント互換性 | 多くの OpenAI-compatible クライアントで利用可能 | サービスごとの設定が必要 |
| モデル切替 | モデル名を変えて利用可能モデルをテスト | endpoint やパラメータ変更が必要な場合あり |
| 利用ログ | コンソールで集中確認 | 複数の管理画面に分散 |
| ドキュメント上のテスト | 現在の endpoint をページ上で直接テスト | Postman や独自コードが必要になりがち |

## 現在の範囲

| 機能 | 状態 | 説明 |
| --- | --- | --- |
| テキストチャット | 対応 | `/v1/chat/completions`。 |
| ストリーミング | 対応 | `stream: true` で SSE を読む。 |
| 関数呼び出し / JSON | モデル依存 | `tools` と `response_format` の対応が必要。 |
| Responses API | 対応 | `/v1/responses`。 |
| Embeddings | 対応 | RAG、検索、類似度。 |
| Claude ネイティブ | モデル依存 | `/v1/messages`、通常 Claude モデル向け。 |
| Gemini ネイティブ | モデル依存 | `/v1beta/models/{model}:generateContent`、通常 Gemini モデル向け。 |
