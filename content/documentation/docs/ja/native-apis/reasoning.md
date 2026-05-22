推論モデルは reasoning_content、reasoning、thinking、または <think>...</think> を返す場合があります。フィールド名はモデルとルートによって異なります。

| フィールド | 可能な出所 | 処理 |
| --- | --- | --- |
| delta.reasoning_content | DeepSeek など | 別表示または折りたたみ。 |
| reasoning / thinking | Gemini または互換ルート | 全モデルにあるとは仮定しない。 |
| <think>...</think> | 一部互換ルート | 表示しない場合は UI で除去。 |
| delta.content | 最終回答 | ユーザー向け本文。 |
