最初のテキストモデル呼び出しは次の順序で行います。

- Aivrae コンソールに登録してログインします。
- 残高またはプリペイドクレジットを確認します。
- API キーを作成します。
- ルーティンググループがある場合は推奨グループから始めます。
- Base URL を https://aivrae.com/v1 に設定し、モデル名を指定して送信します。

```
{
  "base_url": "https://aivrae.com/v1",
  "api_key": "YOUR_AIVRAE_API_KEY",
  "model": "MODEL_NAME"
}
```
