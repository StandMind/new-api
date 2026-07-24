# 云雾价格换算

## 数据源

价格源：

```text
https://yunwu.ai/api/pricing_new
```

每次正式同步重新抓取 JSON，保存 UTC/本地抓取时间与 SHA256。验证响应：

- 根对象；
- `success=true`；
- `data` 为模型数组；
- `group_ratio` 为对象；
- 每个候选有精确 `model_name`。

不要把价格目录出现当作可用性证明。

## 分组选择

需要三组数据：

1. 目标渠道 Key 实际允许的分组名称；
2. 模型 `enable_groups`；
3. 全局 `group_ratio` 和可选 `group_model_ratio`。

对模型 `m` 和分组 `g`：

```text
effective_ratio(m, g) =
  group_model_ratio[g][m]  如果存在模型级覆盖
  group_ratio[g]           否则
```

候选分组：

```text
key_groups ∩ model.enable_groups
```

从候选中选择 `effective_ratio` 最大的分组。相同倍率时保留选择文件中靠前的分组并在审计中记录。

不得：

- 使用 Key 不拥有的分组；
- 使用模型未启用的分组；
- 忽略模型级倍率覆盖；
- 把一个模型选出的分组无条件套给其他模型。

## 售价公式

默认商业参数：

```text
points_per_cny = 2
cny_per_usd = 6.9
markup_multiplier = 1.3
sale_usd = upstream_points / points_per_cny / cny_per_usd * markup_multiplier
```

含义：积分除以 2 得人民币成本，除以 6.9 得美元成本，再加价 30%。

### 普通按 Token 模型

云雾基础字段：

```text
input_points_per_1m = 2 * model_ratio * effective_group_ratio
output_points_per_1m = input_points_per_1m * completion_ratio
```

换算：

```text
input_sale_usd_per_1m = sale_usd(input_points_per_1m)
output_sale_usd_per_1m = sale_usd(output_points_per_1m)
Aivrae ModelRatio = input_sale_usd_per_1m / 2
Aivrae CompletionRatio = output_sale_usd_per_1m / input_sale_usd_per_1m
```

Aivrae 普通 ratio 的基准是 `$2 / 1M input tokens`，因此只有 `ModelRatio` 使用 `/2`。`CompletionRatio` 是输出/输入售价比。

`CacheRatio` 等相对倍率可在确认云雾字段语义后保留。云雾 `audio_ratio` 是绝对音频价格，不是相对文本倍率，不能直接复制。
云雾的 `audio_ratio=0` 或 `audio_completion_ratio=0` 表示未配置，编译时应忽略；若只有正数音频输出价而没有正数音频输入价则停止。

### 按次模型

当前云雾固定价格字段：

```text
points_per_request = model_price * effective_group_ratio
sale_usd_per_request = sale_usd(points_per_request)
Aivrae ModelPrice = sale_usd_per_request
```

必须确认一请求是否恒定对应一次扣费。图像数量、分辨率、视频时长或任务数量改变扣费时，禁止使用单一 `ModelPrice`。

## 必须停止自动换算的情况

- `step_ratios` 非空；
- 思考与非思考 Token 有不同倍率；
- 有 1 小时缓存创建价格；
- 音频字段语义不完整；
- 按次模型价格随 `n`、分辨率、时长或请求参数变化；
- 上游字段缺失、非数字、NaN、无穷或负值；
- 模型与 Key 没有共同分组。

这些情况需要读取 `pkg/billingexpr/expr.md`，按真实美元/百万 Token 价格建立表达式，并验证预扣费、结算和日志。

## 审核

价格候选至少同时展示：

- 云雾模型 ID；
- 选择分组与有效倍率；
- 原始 model ratio/price；
- 输入、输出或单次积分；
- 人民币成本；
- 未加价美元成本；
- 最终美元售价；
- Aivrae option 字段；
- 官方 Standard 价格和核验日期；
- 相对官方价格是折扣还是溢价。

营销折扣只比较相同计费单位和相同档位。Aivrae 按次、官方按 Token/分辨率时标记“计费口径不同”，不制造统一百分比。
