# 分组模型计费与渠道链

本文档说明按模型配置分组倍率、由用户配置有序分组链，以及由管理员为每个
`(分组, 模型)` 独立配置渠道链的使用方式、运行语义和升级注意事项。

## 功能概览

一次请求的路由关系如下：

```text
令牌分组链
vip -> default -> backup
 |        |          |
 |        |          `- gpt-4o: 渠道 6 -> 渠道 7
 |        `- gpt-4o: 渠道 3/4 -> 渠道 5
 `- gpt-4o: 渠道 1/2 -> 渠道 3
```

- 用户只能排列自己有权使用的分组，不能查看或指定分组内的具体渠道。
- 管理员可以为每个 `(分组, 模型)` 配置独立的优先级层级和同层渠道权重。
- 同一渠道可以出现在不同分组、不同模型的路线中，并使用不同的优先级和权重。
- 请求实际在哪个分组成功，就按该分组针对原始请求模型解析出的倍率结算。
- 多组链没有数字形式的总尝试上限，但候选集合是有限的，且每个候选在一次请求中
  最多调用一次，因此不会无限循环。

文中的几个关键概念：

- **用户组**：用户账号自身所属的组。
- **使用分组**：本次上游尝试当前使用的组，可能随分组链切换。
- **分组链**：令牌配置的有序使用分组列表。
- **Ability**：某个渠道对指定分组和模型提供服务的能力记录。
- **显式路线**：管理员保存到 `group_model_routes` 的路线。
- **继承路线**：没有显式路线时，根据 Ability 的 `Priority` 和 `Weight` 动态生成的路线。

## 分组模型倍率

新增系统配置项 `GroupModelRatio`，JSON 结构为：

```json
{
  "vip": {
    "gpt-4o": 0.8,
    "gpt-4*": 0.9,
    "*": 1.0
  },
  "backup": {
    "gpt-4o": 1.2
  }
}
```

第一层键是使用分组，第二层键是原始请求模型或尾部 `*` 前缀通配符。倍率必须是
大于或等于 `0` 的有限数值。通配符只能出现在模式末尾；`*` 可以作为全模型兜底。

倍率按以下固定顺序解析，命中后不再继续回退：

1. `GroupModelRatio[使用分组][原始请求模型]` 精确匹配。
2. `GroupModelRatio[使用分组]` 中匹配原始请求模型的最长前缀通配符。
3. `GroupGroupRatio[用户组][使用分组]`。
4. 普通 `GroupRatio[使用分组]`。

例如 `gpt-4o` 同时匹配 `gpt-*` 和 `gpt-4*` 时，使用更长的 `gpt-4*`。

`GET /api/pricing` 新增两个字段：

- `group_model_ratio`：当前用户可用分组的原始配置，结构为
  `分组 -> 模型模式 -> 倍率`。
- `effective_group_model_ratio`：每个定价模型解析后的有效倍率，结构为
  `模型 -> 分组 -> 倍率`。

每个 `data` 模型项同时包含 `effective_group_ratio`，结构为
`分组 -> 该模型最终有效倍率`。模型广场的 Default 与 Classic 前端统一使用该模型级
结果：

- 选择具体分组时，只展示该模型在所选分组下的价格。
- 选择“全部分组”时，每个模型分别使用自身可用分组中的最低倍率展示价格。
- 模型详情用一张表列出该模型对当前用户可用的全部分组、有效倍率和对应价格，并按
  有效倍率升序、分组名稳定排序。
- 按量、固定价格、缓存、图片、音频和 tiered expression 都使用相同的模型级倍率。
- 阶梯模型按“分组 × 档位”生成表格行；只有多档模型显示“档位”列，不再为每个分组
  单独生成价格子表。
- 无法解析为标准档位的特殊表达式保留原文警告，不生成推测价格。
- 分组筛选只显示组名，不展示全局倍率，因为同一分组对不同模型可能具有不同倍率。

前端按分组键合并倍率，优先级为模型项的 `effective_group_ratio`、顶层
`effective_group_model_ratio[模型]`、普通 `group_ratio`，最终才回退为 `1`。这样在
前端先于后端升级时仍兼容旧 `/api/pricing` 响应；倍率 `0` 是合法免费配置，不会被
当成缺失值。

## 令牌分组链

Token 表新增 `group_chain TEXT`，令牌 API 新增：

```json
{
  "group": "vip",
  "group_chain": ["vip", "default", "backup"]
}
```

规则如下：

- `group_chain` 永久启用且必须至少包含一个分组；不再存在功能开关。
- `group` 始终保存 `group_chain` 的首组，用于首组限流及现有单组字段读取。
- 数据库中没有显式 `group_chain` 的令牌无效，不会回退到 `group`。
- `group=auto` 或链中包含 `auto` 的令牌直接拒绝，不做转换或自动展开。
- `AutoGroups`、`DefaultUseAutoGroup` 和
  `routing_setting.user_group_chain_enabled` 已从配置与接口中删除。
- 新用户的默认令牌使用用户当前组作为单组显式链。
- 模型列表返回令牌各分组可用模型的并集。
- 分组限流仍只按令牌首组计数一次，不会因后备组尝试重复计数。

分组链写入时执行以下校验：

- 不允许空分组。
- 不允许重复分组。
- 不允许在显式链中包含 `auto`。
- 每个分组都必须属于用户当前可用组。
- 每个分组都必须仍存在于 `GroupRatio` 配置中。
- 不设置固定链长上限。

跨组重试是分组链的固定路由行为，不是令牌级用户选项。当前组内候选耗尽后，系统会
自动进入下一组；如果当前组没有可用渠道、渠道已禁用、没有可用密钥或渠道初始化失败，
也会直接继续查找下一组。

Default 与 Classic 的 API Key 编辑器均使用可搜索多选下拉框。新选分组按选择顺序追加，
下方有优先级编号和拖拽手柄，可通过鼠标、触摸或键盘排序；取消选择会从链中移除，但
至少保留一个分组。提交时始终发送完整 `group_chain` 并同步首组 `group`。

## 官方参考价

管理员可通过 Option `official_price_setting.model_prices` 配置模型的官方参考价。该配置
只用于模型广场价格比较，不参与预扣、结算或计费日志。结构示例：

```json
{
  "gpt-5.6-sol": {
    "unit": "usd_per_million_input_tokens",
    "source_model": "gpt-5.6-sol",
    "source_url": "https://developers.openai.com/api/docs/pricing",
    "verified_at": "2026-07-30",
    "tiers": [
      {"up_to_input_tokens": 272000, "price": 5},
      {"price": 10}
    ]
  }
}
```

配置校验规则：

- `unit` 只允许 `usd_per_million_input_tokens` 和 `usd_per_request`。
- 价格必须为大于 `0` 的有限数；按次参考价只能包含一个档位。
- Token 阶梯上限必须为严格递增的正整数，最后一档不得设置上限。
- `source_url` 必须是有效的 HTTP(S) URL，`verified_at` 必须使用 `YYYY-MM-DD`。

`GET /api/pricing` 在对应模型项上增量返回 `official_price`；没有配置时省略。模型详情
使用未经过币种、充值汇率转换的实际 API 售价与同档输入参考价比较，并在分组名称后显示
“比官方便宜 X%”“与官方同价”或“高于官方 X%”。百分比保留一位后移除无意义的 `.0`；
免费倍率显示便宜 `100%`，单位或档位无法匹配时不显示标签。标签 tooltip 展示官方基准价、
来源型号、核验日期和可访问的来源链接。

管理后台的“官方参考价”页支持搜索、新增、编辑和删除。Default 使用右侧抽屉，Classic
使用大尺寸弹窗；两者都编辑整个 JSON Option，不增加数据库表。

当前 OpenLux 分组导入沿用 26 条 2026-07-30 参考价，来源为 OpenAI API Pricing 和
Gemini API Pricing。`gpt-5.2-chat` 映射官方 `gpt-5.2-chat-latest`；
`gpt-5.3-codex-spark` 无公开标准 API 单价，`gemini-3-pro-preview` 已关闭，因此两者不
初始化，也不显示比较标签。

## 分组模型渠道链

数据库新增 `group_model_routes` 表，以 `group + model` 为联合主键。`tiers` 以 JSON
保存在文本列中，示例：

```json
{
  "group": "vip",
  "model": "gpt-4o",
  "tiers": [
    {
      "priority": 100,
      "channels": [
        {
          "channel_id": 1,
          "weight": 100
        },
        {
          "channel_id": 2,
          "weight": 50
        }
      ]
    },
    {
      "priority": 50,
      "channels": [
        {
          "channel_id": 3,
          "weight": 100
        }
      ]
    }
  ]
}
```

优先级数值越大越先执行，同一优先级内按权重选择。路线保存时会校验：

- 分组、模型不能为空，且至少有一个优先级层级。
- 同一路线内优先级不能重复，每层至少有一个渠道。
- 同一路线内渠道不能重复。
- `channel_id` 必须大于 `0`。
- 权重必须在 `0` 到 `2147483647` 之间。
- 渠道必须启用，并具有该分组和模型对应的启用 Ability。

权重 `0` 不表示禁用渠道。当同层还有正权重渠道时，零权重渠道不会先被抽中；正权重
渠道耗尽后，剩余零权重渠道仍会按随机顺序进入尝试列表。要从路线中禁用某个渠道，
应将它从该路线移除或禁用对应 Ability。

没有显式路线时，系统根据 Ability 的 `Priority` 和 `Weight` 生成继承路线。内存缓存
开启和关闭时都使用 Ability 或显式路线上的优先级和权重，不再读取 Channel 的全局值。

## 管理接口

路线接口都位于 `/api/group-model-routes`，仅管理员可以访问。保存和删除操作会写入
管理审计日志。

### 列出已保存路线

```http
GET /api/group-model-routes/list
```

返回全部显式路线，按最近更新时间优先排列。该接口用于管理员前端展示已有路线；每条
记录包含 `group`、`model`、完整 `tiers` 和 `updated_at`。继承 Ability 但未显式保存
的路线不会出现在列表中。

列表中的渠道在原有 `channel_id` 和 `weight` 外增加两个只读展示字段：

```json
{
  "channel_id": 1,
  "channel_name": "OpenAI 主渠道",
  "weight": 100,
  "eligible": true
}
```

- `channel_name` 从当前 Channel 批量解析，不写入路线配置。
- `eligible=true` 表示渠道当前已启用，并仍有对应 `(分组, 模型)` 的已启用 Ability。
- 已禁用、Ability 已移除或 Channel 已删除的配置引用仍会返回；此时
  `eligible=false`，Channel 已删除时 `channel_name` 为空。
- 列表展示元数据通过批量查询生成，不会为每条路线单独查询 Channel 或 Ability。

### 查询路线

```http
GET /api/group-model-routes?group=vip&model=gpt-4o
```

响应 `data`：

```json
{
  "explicit": true,
  "route": {
    "group": "vip",
    "model": "gpt-4o",
    "tiers": [
      {
        "priority": 100,
        "channels": [
          {
            "channel_id": 1,
            "weight": 100
          }
        ]
      }
    ]
  },
  "candidates": [
    {
      "channel_id": 1,
      "channel_name": "channel-1",
      "priority": 100,
      "weight": 100
    }
  ]
}
```

`explicit=false` 时，`route.tiers` 是根据当前 Ability 生成的可编辑继承结果。
`candidates` 列出当前确实可用于该分组和模型的启用渠道。

### 保存路线

```http
PUT /api/group-model-routes
Content-Type: application/json
```

请求体使用完整的 `group`、`model` 和 `tiers` 结构。该操作是全量保存，不是局部合并；
客户端必须提交希望保留的所有层级和渠道。

### 恢复继承路线

```http
DELETE /api/group-model-routes?group=vip&model=gpt-4o
```

删除显式路线后，该 `(分组, 模型)` 立即恢复使用 Ability 的 `Priority` 和 `Weight`。
响应中的 `deleted` 表示是否实际删除了记录。

## 路由与重试语义

请求开始时，系统生成不可变的 `RouteAttemptPlan`：

```text
用户分组顺序
  -> 各分组的模型路线
    -> Priority 从高到低
      -> 同层渠道按权重生成不放回顺序
```

计划生成后，本次请求不会因中途修改配置而重排。每个
`(分组, 路由模型, 渠道)` 在一次请求中最多尝试一次：

1. 先尝试当前层按权重选出的渠道。
2. 发生可重试错误时，尝试同层尚未使用的渠道。
3. 当前层耗尽后进入下一优先级。
4. 当前组耗尽后进入下一组。
5. 所有允许的有限候选耗尽后结束。

`RetryTimes` 只保留给单组显式链：

| 令牌形式 | 是否受 `RetryTimes + 1` 总上游尝试次数限制 |
| --- | --- |
| 只含一个组的显式 `group_chain` | 是 |
| 包含多个组的显式 `group_chain` | 否，遍历有限候选集 |
| 空链或 `group=auto` | 不执行路由，认证阶段直接拒绝 |

这里的“没有总尝试次数限制”不是无限重试。系统不会重新使用已经尝试过的候选；最终
一定会在路线候选全部耗尽时终止。同一渠道如果分别属于两个分组，则属于两个不同候选，
可能在每个分组中各尝试一次。

以下情况会提前停止或跳过：

- 非重试型 `4xx` 上游错误立即停止。
- 流式响应已经输出任何内容后停止，避免拼接两个上游响应。
- 管理员指定渠道的请求失败后立即停止。
- 渠道不可用、无可用密钥或初始化失败时跳过，不产生上游调用；指定渠道除外。
- 异步任务只在上游明确未接单的 `429` 或配置为可重试的 `5xx` 响应下继续。
- 异步任务发生网络超时、接单状态不确定、本地错误或渠道锁定时停止，避免重复创建任务。

## 计费语义

路由每次首次进入一个新分组、并在调用该组首个渠道之前，都会使用原始请求模型重新
解析分组倍率和预扣额度。

- 首组免费、后备组收费：进入收费组前创建 `BillingSession` 并预扣。
- 后备组更贵：调用上游前通过 `BillingSession.Reserve()` 补足预扣。
- 补扣失败：不调用该组上游，直接返回额度不足。
- 后备组更便宜：最终按成功组实际额度结算，多预扣部分统一退还。
- 请求失败：退还尚未结算的预扣额度。

最终计费使用实际成功分组的倍率，而不是令牌首组的倍率。按量计费、固定价格、
tiered expression、实时音频和异步任务均遵循该规则：

- tiered expression 在切组时重新按新倍率计算预扣，并以最终成功组结算。
- 实时音频结算时重新解析最终使用分组的倍率。
- 异步任务提交时按当前尝试分组预扣；任务记录保存成功分组的倍率快照，后续配置变更
  不会重定价已经在途的任务。

## 日志与审计

普通路由日志的 `other` 中新增：

- `group_chain`：请求配置的分组链。
- `attempted_groups`：实际产生上游尝试的分组顺序。
- `final_group`：最终使用分组。
- `group_ratio_source`：倍率来源，例如 `group_model_ratio.exact`、
  `group_model_ratio.prefix`、`group_group_ratio` 或 `group_ratio`。

具体渠道信息仅写入管理员信息：

```text
other.admin_info.routing.planned
other.admin_info.routing.attempted
```

每个记录包含分组、模型、优先级、渠道 ID，以及是否来自显式路线。用户侧日志不暴露
具体渠道。管理员保存或删除路线时分别记录 `group_model_route.update` 和
`group_model_route.delete` 管理审计。

## 前端入口

Default 前端：

- 系统设置中的分组与模型定价页面可编辑 `GroupModelRatio`、分组模型渠道链和官方
  参考价。
- 渠道链页签打开后直接展示全部已保存显式路线，不需要先选择分组和模型。
- 路线表格支持按分组筛选、按分组/模型/渠道搜索、同时展开多条路线，以及全部展开或
  收起；展开行直接显示完整优先级层级、渠道名称、渠道 ID、权重和当前可用状态。
- 编辑已有路线使用右侧抽屉，分组和模型不可修改；仅在新建路线时选择分组和模型，
  选择完成后自动加载继承 Ability 路线。保存后列表自动刷新并展开对应行。
- 恢复继承路线和放弃未保存修改均需要确认。
- API Key 新建/编辑抽屉提供可搜索多选和拖拽排序；跨组重试固定启用，不向用户提供开关。
- 模型详情把普通、固定价和阶梯计费统一为一张分组价格表，并显示官方价格比较标签。
- 渠道链编辑器实现位于
  `web/default/src/features/system-settings/models/group-model-route-editor.tsx`。

Classic 前端：

- 系统设置的分组倍率页新增“分组模型渠道链”页签。
- 渠道链页签使用与 Default 相同的可搜索、可筛选、可多行展开路线总览。
- 编辑和新建在大尺寸弹窗中完成；已有路线无需重新选择分组和模型。
- 失效渠道不会从显式配置中隐藏，列表会标记不可用，编辑器允许将其移除或替换。
- Token 新建/编辑弹窗提供可搜索多选和拖拽排序；跨组重试固定启用，不向用户提供开关。
- 模型详情与 Default 使用相同的分组排序、阶梯行和官方价格比较逻辑。
- 渠道链编辑器实现位于
  `web/classic/src/pages/Setting/Ratio/GroupModelRouteSettings.jsx`。

普通用户只能看到可选分组及其顺序，看不到管理员配置的候选渠道、优先级或权重。

## 部署步骤

该版本不保留旧自动分组和空链令牌兼容逻辑，部署前按以下顺序处理：

1. 备份数据库，部署包含 `group_chain` 和 `group_model_routes` 的版本。
2. 确保所有需要保留的令牌都有非空显式 `group_chain`，且 `group` 等于链首组。
3. 删除三个废弃自动分组 Option，配置 `GroupModelRatio` 和官方参考价。
4. 按需配置各 `(分组, 模型)` 的显式渠道链，并验证继承路线和显式路线。
5. 创建单组与多组测试令牌，验证失败切组、补扣、退款、最终日志和官方价展示。

删除显式路线是可恢复继承行为的配置操作，不会删除 Channel 或 Ability。

## OpenLux 上游分组导入

`work/aivrae-openlux/` 保存本次 OpenLux 价格准备、账号分组探测、生产导入和验收工具。
该目录被 Git 忽略；系统访问令牌只通过终端提示或标准输入进入进程，不得写入命令
参数、环境、清单、报告或文档。OpenLux 管理接口还要求与令牌所属账号匹配的数字型
`New-Api-User`。

```bash
python3 work/aivrae-openlux/prepare_expansion.py prepare
python3 work/aivrae-openlux/prepare_expansion.py verify
go run ./work/aivrae-openlux/validate_expansion_billing
python3 work/aivrae-openlux/expand_production.py --dry-run
python3 work/aivrae-openlux/expand_production.py \
  --execute --confirm-model-count 298 --confirm-group-count 58 \
  --openlux-user-id <uid>
python3 work/aivrae-openlux/verify_expansion.py
```

`prepare_expansion` 读取 Aivrae 与 OpenLux 的公开 `/api/pricing`，保存抓取时间、原始
响应和 SHA256。导入前的候选范围是 449 个 OpenLux 公开模型；按账号可创建分组、当前
代码已有协议和端点适配器、模型未在 Aivrae 启用等条件筛出 298 个候选模型、58 个分组
和 1350 个 `(分组, 模型)` 组合。77 个端点协议尚不支持、46 个模型没有账号可用分组、
28 个模型已存在，因此不会重复导入。

OpenLux 使用美元计价，不做人民币或积分换算。Aivrae 对每个精确 `(分组, 模型)` 保存
模型级分组倍率，使标准调用价格等于 OpenLux 同组合公开价的 1.3 倍：

```text
Token 模型 GroupModelRatio
  = OpenLux ModelRatio × OpenLux GroupRatio × 1.3
    ÷ Aivrae ModelRatio

固定价格模型 GroupModelRatio
  = OpenLux ModelPrice × OpenLux GroupRatio × 1.3
    ÷ Aivrae ModelPrice
```

所有计算使用十进制定点数，中间不提前舍入，配置最多保留 15 位小数。输出、缓存、缓存
写入、图片和音频沿用 OpenLux 对应的相对倍率。29 个可安全复现的阶梯模型使用
`tiered_expr`，档位条件同时覆盖输入和输出阈值；最终再乘实际成功分组的精确模型倍率。

OpenLux 另有 11 个 Qwen 模型同时给出非思考和思考两套阶梯倍率：
`qwen-plus`、`qwen-plus-2025-12-01`、`qwen-plus-latest`、`qwen3-0.6b`、
`qwen3-1.7b`、`qwen3-14b`、`qwen3-235b-a22b`、`qwen3-30b-a3b`、
`qwen3-32b`、`qwen3-4b`、`qwen3-8b`。实际探测中，仅传 `enable_thinking=true` 但没有
返回思考内容时，OpenLux 仍使用非思考倍率；因此不能只根据请求参数预测上游最终档位。
当前表达式又没有响应侧思考状态变量，无法可靠复现，所以这 11 个模型和对应 50 条
Ability 保持禁用，不能用普通输出倍率代替思考倍率。

正式导入按以下顺序执行：校验发布槽和并发指纹，创建 OpenLux 永久单组 Key，备份并
校验 PostgreSQL，创建禁用渠道，刷新渠道缓存，写入倍率和阶梯配置，启用渠道，最后
更新 `UserUsableGroups`。每个渠道的模型列表严格等于该组验证后清单，Priority 为 0、
Weight 为 100；当前没有显式 `group_model_routes`，因此使用 Ability 继承路线。

最终生产开放 287 个本次新增模型：240 个文本、19 个图片、17 个音频、6 个 embedding
和 5 个 rerank；对应 58 个 OpenLux 单组渠道、58 个单组上游 Key、1300 条启用 Ability
和 1300 条精确模型级分组倍率。加上原有 `default`，用户可选分组总数为 59；公开模型
总数为 316。

验收覆盖 OpenAI Chat、Responses、Embedding、Rerank、图片、TTS、Gemini 原生、
Anthropic 原生，以及真实跨组回退。`Azure-Gpt-1 -> Openai-Gpt-1` 在首组真实返回可
重试 503 后进入第二组成功；另一次无 Ability 路径使用 `SiliconFlow-1 -> Azure-Gpt-2`
成功。固定价 TTS 在上游成功但不返回 token usage 时仍按 `ModelPrice × 成功分组倍率`
结算，普通音频、WSS 和文本路径均有回归测试。上游返回 429、模型不存在、渠道不可用
或鉴权失败的组合只记为外部可用性跳过，不视为本地路由成功。

逐项价格审计要求 1300 个启用组合均为 OpenLux 同组合价格的 1.3 倍；完整上游 Key 只
存在于 OpenLux 和生产 Channel 密钥字段，不出现在 manifest、报告、日志、文档或 Git。
生产不存在 `Yunwu/` 渠道，所有临时系统访问令牌和冒烟 Key 在验证结束后清理。

## 实现索引

- 倍率解析：`setting/ratio_setting/group_ratio.go`
- 令牌分组链校验：`controller/token.go`
- 路由表与保存校验：`model/group_model_route.go`
- 管理接口：`controller/group_model_route.go`
- 不放回尝试计划：`service/route_attempt_plan.go`
- 请求重试与切组：`controller/relay.go`
- 预扣、补扣与结算：`service/billing_session.go`
- 路由日志：`service/route_logging.go`

## 验收清单

- [ ] 同一渠道在不同 `(分组, 模型)` 路线中可使用不同优先级和权重。
- [ ] 显式路线保存后生效，删除后恢复 Ability 路线。
- [ ] 内存缓存开启和关闭时，候选优先级和权重一致。
- [ ] 同层渠道按权重不放回排序，每个候选最多调用一次。
- [ ] 多组候选数超过 `RetryTimes + 1` 时仍继续尝试并最终终止。
- [ ] 单组显式链仍受 `RetryTimes + 1` 限制。
- [ ] 空链和 `group=auto` 令牌在认证阶段被拒绝。
- [ ] A 组全部失败后进入 B 组成功，并只按 B 组模型倍率结算。
- [ ] 当前组候选耗尽后会固定进入下一组，不存在令牌级关闭选项。
- [ ] 首组免费、后备收费时能够创建计费会话。
- [ ] 更贵后备组补扣失败时不会调用上游。
- [ ] 更便宜后备组成功后会退还多预扣额度。
- [ ] tiered expression、固定价格、按量、实时音频和异步任务使用最终成功组倍率。
- [ ] 流式响应已有输出时不再重试。
- [ ] 异步任务只重试明确未接单的安全错误。
- [ ] 模型详情按倍率排序，普通、固定和阶梯价格使用同一张表。
- [ ] 官方价配置通过校验并显示正确的便宜、同价、高于官方或无标签状态。
- [ ] Default 前端通过 typecheck、lint 和 build。
- [ ] Classic 前端通过 lint 和 build。
- [ ] Go 后端全量测试通过。
- [ ] SQLite、MySQL、PostgreSQL 的 Token 和 `group_model_routes` 迁移通过。

## 第一期边界

当前路线完全由管理员显式配置或继承 Ability 配置。第一期不包含：

- 按价格、延迟、速度或成功率自动排序。
- 根据实时健康度自动改写持久化路线。
- 普通用户查看或指定具体渠道。
- 请求级路由后缀或临时覆盖路线。
- 数字形式的多组总重试上限。

调整权重只影响同一优先级层内的尝试顺序，不代表调用配额，也不保证短时间窗口内严格
符合权重比例。若要停止使用某个渠道，应从显式路线移除它或禁用对应 Ability。
