# 同步清单规范

使用 `assets/sync-manifest.template.json` 创建每次同步清单。清单是审计记录和并发保护，不是完整数据库导出。

## 顶层字段

| 字段 | 要求 |
| --- | --- |
| `schema` | 固定为 `aivrae-model-sync/v1` |
| `generated_at` | 带时区的 ISO 8601 时间 |
| `channel_id` | 从当前环境读取的正整数 |
| `channel_name` | 与 API 返回精确匹配 |
| `expected_original_models` | 当前渠道模型的有序数组，不排序 |
| `remove_models` | 用户明确要求移除的精确模型 ID |
| `add_models` | 用户明确选择新增的精确模型 ID |
| `source` | 上游、快照哈希、分组和公式参数 |
| `source_pricing` | 可重算价格的逐模型输入与结果 |
| `managed_option_keys` | 本次脚本允许读取和修改的 option map |
| `option_patch` | 只包含新增模型的局部价格补丁 |
| `model_metadata` | 每个新增模型一条元数据 |
| `route_tests` | 每个新增模型的生产路由测试 |

`add_models` 与 `remove_models` 必须唯一且互斥。新增模型不能已出现在 `expected_original_models` 中，移除模型通常必须出现在其中。
仅移除的清单可以将 `source.token_groups`、`source_pricing`、`option_patch`、`model_metadata` 和 `route_tests` 设为空集合；仍需保留上游快照、公式参数和移除依据。
顶层字段必须与模板一致，重复 JSON 键或额外顶层字段会被拒绝。

## source_pricing

普通按 Token 模型必须保存可完整重算的字段：

```json
{
  "model-name": {
    "quota_type": 0,
    "selected_group": "group-name",
    "selected_group_ratio": 1,
    "group_ratio_source": "group_ratio",
    "model_ratio": 1,
    "completion_ratio": 4,
    "input_points_per_1m": 2,
    "output_points_per_1m": 8,
    "input_cost_cny_per_1m": 1,
    "output_cost_cny_per_1m": 4,
    "input_cost_usd_per_1m": 0.14492753623188406,
    "output_cost_usd_per_1m": 0.5797101449275363,
    "input_sale_usd_per_1m": 0.18840579710144928,
    "output_sale_usd_per_1m": 0.7536231884057971
  }
}
```

按次模型必须保存：

```json
{
  "model-name": {
    "quota_type": 1,
    "selected_group": "group-name",
    "selected_group_ratio": 1,
    "group_ratio_source": "group_ratio",
    "model_price": 0.1,
    "points_per_request": 0.1,
    "cost_cny_per_request": 0.05,
    "cost_usd_per_request": 0.007246376811594203,
    "sale_usd_per_request": 0.009420289855072464
  }
}
```

`group_ratio_source` 只能是全局 `group_ratio` 或模型级
`model_override`。复杂计费额外记录阶梯、模态、请求参数或时长依据，
并为普通 ratio 回退保留上述可重算字段。不能只保存最终价格。

## option_patch

`option_patch` 是局部补丁，不能直接替换完整 options。

- 普通按 Token：`ModelRatio`、`CompletionRatio`，以及存在时的 `CacheRatio`、`CreateCacheRatio`、`ImageRatio`、`AudioRatio`、`AudioCompletionRatio`。
- 按次：`ModelPrice`。
- 阶梯/动态：`billing_setting.billing_mode` 为 `tiered_expr`，并设置 `billing_setting.billing_expr`。表达式可能同时保留普通 ratio 作为兼容回退。

每个新增模型必须有一个基础计费方式。不得同时把同一模型配置为普通 ratio 和固定 `ModelPrice`。
`managed_option_keys` 必须包含模板中的完整集合，使移除操作不会遗留旧计费键。

移除模型不放在 `option_patch`。应用脚本从所有 `managed_option_keys` 中删除它。

## model_metadata

每条元数据包含：

```json
{
  "model_name": "exact-model-id",
  "description": "English description",
  "description_i18n": {
    "en": "",
    "zh": "",
    "es": "",
    "fr": "",
    "ru": "",
    "ja": "",
    "vi": ""
  },
  "icon": "Gemini",
  "tags": "chat,vision,tools",
  "tags_i18n": {
    "en": "",
    "zh": "",
    "es": "",
    "fr": "",
    "ru": "",
    "ja": "",
    "vi": ""
  },
  "vendor_id": 0,
  "endpoints": "",
  "status": 1,
  "sync_official": 0,
  "name_rule": 0
}
```

- 从当前 Aivrae vendor API 或数据库读取 `vendor_id`，不按历史值猜测。
- `endpoints` 是 JSON 字符串或空字符串。非空时每项必须使用项目已知 endpoint type，并且值只包含绝对 `path` 和大写 HTTP `method`。只填经过代码和实测确认的 endpoint map。
- 描述不得声称未验证的官方合作、能力或稳定性。
- 七种语言都必须非空且语义一致。

## route_tests

每个测试项格式：

```json
{
  "model": "exact-model-id",
  "endpoint_type": "openai",
  "allowed_failure_classifications": []
}
```

`endpoint_type` 使用项目当前 endpoint 类型，例如 `openai`、`gemini`、`embeddings`、`image-generation` 或 `openai-response`。必须从代码与实测确认。

只有用户明确接受的临时上游问题才能放入 `allowed_failure_classifications`。支持的分类为：

- `upstream_capacity`

不要把 `model_not_found`、鉴权失败、转换错误或普通 `5xx` 加入允许列表。
允许的 `upstream_capacity` 仍表示“尚未验证可用”，不能在报告中改写成成功。

## 禁止内容

清单及其旁路文件不得包含：

- `Authorization` header；
- 系统访问令牌或 API Key；
- 渠道完整 Key；
- 数据库、Redis 或 SSH 凭据；
- 临时 root token；
- Cookie 或 session。

运行 `scripts/validate_manifest.py` 检查结构、公式、元数据集合和常见 secret 模式。
