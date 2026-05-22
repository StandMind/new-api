Reasoning models may return reasoning_content, reasoning, thinking, or <think>...</think> wrapped content. Field names vary by model route and model.

| Field / form | Possible source | Handling |
| --- | --- | --- |
| delta.reasoning_content | Reasoning models such as DeepSeek | Show separately or collapse it. |
| reasoning / thinking | Gemini or compatible routes | Do not assume every model has it. |
| <think>...</think> | Some compatible or reverse routes | Filter in the UI if you do not show reasoning. |
| delta.content | Final answer content | Primary user-facing text. |
