モデル名はコンソールと完全一致する必要があります。アカウント、グループ、プランによって見えるモデルは異なります。

**Endpoint:** `GET /v1/models`

| フィールド | 説明 |
| --- | --- |
| id | model フィールドに渡す値。 |
| owned_by / provider | 提供元またはルート種別。 |
| コンテキスト長 | コンソールに従います。超過時は 400、413、サービスエラー。 |
| endpoint 互換性 | Chat は chat completions、embedding は embeddings、Claude は /v1/messages、Gemini は generateContent。 |

```
curl https://aivrae.com/v1/models \
  -H "Authorization: Bearer YOUR_AIVRAE_API_KEY"
```
