Gateway price differences usually come from model route, account pool, concurrency, stability, region, and model permissions. If Aivrae exposes groups in the console, the console is the source of truth.

| Group type | Best for | Price / stability notes |
| --- | --- | --- |
| Default group | General tests and normal text chat | Recommended starting point with balanced cost and availability. |
| High-stability group | Production apps and customer-facing services | Usually higher cost with better concurrency and stability. |
| Low-cost group | Non-critical or batch tasks | Lower cost but may have stricter limits or more fluctuation. |
| Claude / Gemini / local-model group | Specific model families | Model families can have separate multipliers and permissions. |

> If the same model has different prices in different groups, that is normal. Use the model page, key page, and usage logs as the source of truth.
