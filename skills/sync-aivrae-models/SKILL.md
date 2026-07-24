---
name: sync-aivrae-models
description: Discover upstream models missing from Aivrae, verify model availability and official status, select the highest effective Yunwu pricing group allowed by the channel key, calculate Aivrae prices, prepare localized metadata and a concurrency-guarded manifest, back up production, apply additions/removals with rollback, and verify public pricing and real routing. Use for Aivrae/new-api model catalog comparison, 云雾模型同步、缺失模型检查、模型增删、上游价格换算、gemini-default 或其他渠道模型配置、生产模型同步、模型价格更新、渠道 ability 验证和同步失败回滚。
---

# Aivrae 上游模型同步

## 核心约束

- 将流程分成 `发现`、`准备`、`执行` 三个阶段。发现和准备不授权生产写入。
- 在用户明确选择新增、移除和目标渠道前停止在候选清单，不替用户扩大模型范围。
- 不把系统访问令牌、API Key、渠道 Key、数据库密码、临时 root token 或完整 secret-bearing 环境变量写入 skill、清单、审计文件、命令输出或回复。
- 将 `work/令牌` 视为秘密文件。只在进程内解析需要的值，不回显内容，不复制到服务器审计目录。
- 先验证上游真实可用性，再同步。模型仅出现在 `/models` 或价格目录中不代表可调用。
- 将上游 `429`、容量饱和或持续 `5xx` 记录为不可用，不把配置成功误报为模型可用。
- 使用上游 Key 实际允许的分组与模型 `enable_groups` 的交集；按每个模型的有效倍率选择最贵分组。不得选全局最高但该 Key 或模型不可用的分组。
- 使用公式 `售价 USD = 云雾积分 / 2 / 6.9 * 1.3`，除非用户明确改变积分兑人民币、汇率或利润率。
- 按 Token、按次、阶梯、图片、音频和任务计费必须分别处理。不能用普通 `ModelRatio` 公式猜测复杂计费。
- 任何 tiered/dynamic billing 变更前完整读取 `pkg/billingexpr/expr.md`，表达式系数使用真实美元/百万 Token 价格，不使用 `/2` ratio 约定。
- 生产写入前必须有完整 PostgreSQL 备份、`gzip -t` 校验、SHA256、审计目录、并发保护和 dry-run。
- API 脚本默认只 dry-run；执行必须同时传 `--execute` 和正确的 `--confirm-channel-id`。
- 保留用户工作树中已有修改，不提交或上传 `work/令牌`、价格原始 Key、生产渠道 Key或临时凭据。

## 读取顺序

1. 始终读取 [references/manifest.md](references/manifest.md)。
2. 上游为云雾时读取 [references/yunwu-pricing.md](references/yunwu-pricing.md)。
3. 需要生产读取或写入时读取 [references/aivrae-production.md](references/aivrae-production.md)。
4. 清单包含 `billing_setting.billing_mode` 或 `billing_setting.billing_expr` 时，再读取 `pkg/billingexpr/expr.md`。
5. 需要连接 `lingyun1` 时调用 `wan-server-environment` skill，使用其中的主机、用户、端口、私钥和 known_hosts，不在本 skill 复制连接秘密。

## 阶段判断

### 发现

适用于“有哪些模型没接入”“查看上游模型”“比较目录”。只执行只读请求并返回候选，不创建清单、不改配置。

### 准备

适用于“计算价格”“准备同步”“我选择这些模型”。获取快照、实测上游、生成价格候选、元数据和 manifest，运行本地校验及 API dry-run，但不执行写入。

### 执行

适用于用户明确要求“添加、移除、同步到生产”。完成备份和审计后应用 manifest，失败时回滚，最后验证公开价格、ability、元数据和真实路由。

## 工作流

### 1. 确认目标

1. 确认站点、环境、渠道名称、上游和用户要新增/移除的精确模型 ID。
2. 对生产环境读取活动槽、容器健康和目标渠道当前状态；不要从历史记录假定活动槽或渠道 ID。
3. 记录当前渠道模型的有序列表。后续将它写入 manifest 的 `expected_original_models`，作为并发变更保护。
4. 若用户只要求查看缺失模型，完成步骤 2 至 4 后返回候选并等待选择。

### 2. 发现并筛选模型

1. 获取带时间戳的上游价格快照，记录 URL、抓取时间和 SHA256。
2. 获取目标渠道当前模型、Aivrae 公开价格和活跃模型元数据。
3. 用精确模型 ID 做集合差，区分：
   - 上游存在、Aivrae 未接入；
   - Aivrae 已接入；
   - 上游已停用或文档不存在；
   - 价格目录存在但真实请求失败。
4. 查阅厂商官方模型页、弃用页和 Standard API 价格页。记录核验日期，不把第三方别名自动当成官方型号。
5. 按模型类型发最小真实请求：
   - 文本模型：非流式短文本请求；
   - 图像模型：原生图像生成请求，并检查响应中确有图像数据；
   - Embedding：真实 embedding 请求，并检查向量非空；
   - 其他媒体/任务模型：提交后追踪到终态。
6. 向用户展示缺失、官方状态、上游实测结果和价格口径，等待用户明确选择。

### 3. 选择云雾分组并计算售价

1. 安全读取目标渠道 Key 实际拥有的分组名称，只记录分组名称，不记录 Key。
2. 对每个模型计算 `Key 分组 ∩ model.enable_groups`。
3. 有 `group_model_ratio[group][model]` 时优先使用模型级倍率，否则使用 `group_ratio[group]`。
4. 从交集中选择有效倍率最高的分组。没有交集时停止，不猜测价格。
5. 从 [assets/pricing-selection.template.json](assets/pricing-selection.template.json) 创建选择文件，填入允许分组和用户选择的模型。
6. 运行：

```bash
python3 skills/sync-aivrae-models/scripts/compile_yunwu_pricing.py \
  --source path/to/yunwu-pricing.json \
  --selection path/to/pricing-selection.json \
  --output path/to/pricing-candidate.json
```

7. 审核每个模型的上游积分、人民币成本、美元成本、售价、利润率和 Aivrae 配置字段。
8. 脚本拒绝的阶梯、思考、音频或其他复杂价格不能人工绕过。读取计费表达式文档，补充正确实现和边界测试后再继续。
9. 按次模型必须额外确认 `n`、分辨率、时长和任务数量是否会改变上游扣费。若会变化，不能使用单一 `ModelPrice`。

### 4. 构建同步清单

1. 从 [assets/sync-manifest.template.json](assets/sync-manifest.template.json) 创建本次 manifest，不修改模板。
2. 合并价格候选的 `source_pricing` 和 `option_patch`。
3. 为每个新增模型填写七语言元数据：`en`、`zh`、`es`、`fr`、`ru`、`ja`、`vi`。
4. 从代码和实测确认 endpoints。Embedding、图像、Responses、原生 Gemini 等端点不能仅凭模型名猜测。
5. 为每个新增模型配置明确的 `route_tests`。图像测试使用原生协议时仍要单独检查生成内容。
6. 不在 manifest 中放凭据。运行：

```bash
python3 skills/sync-aivrae-models/scripts/validate_manifest.py path/to/manifest.json
```

### 5. API dry-run

1. 通过环境变量提供管理认证：

```text
AIVRAE_API_BASE
AIVRAE_ROOT_ID
AIVRAE_ROOT_TOKEN
```

2. 先运行只读预检：

```bash
python3 skills/sync-aivrae-models/scripts/apply_manifest.py \
  path/to/manifest.json
```

3. dry-run 必须确认：
   - 管理权限有效；
   - 渠道 ID、名称、状态匹配；
   - 当前模型有序列表与 manifest 完全一致；
   - 新增元数据尚不存在；
   - 移除目标状态符合预期；
   - 移除目标没有被其他渠道引用；
   - 元数据使用的供应商 ID 当前存在；
   - managed options 均存在且为 JSON 对象。
4. 任一并发检查失败时重新读取当前状态并重建 manifest，不强行覆盖。

### 6. 备份和审计

1. 按 [references/aivrae-production.md](references/aivrae-production.md) 创建完整 PostgreSQL 备份。
2. 验证管道在 `pipefail` 下成功、归档非空、`gzip -t` 通过，并写 SHA256 sidecar。
3. 创建权限为 `0700` 的本次审计目录；脚本、manifest 和结果文件设为 `0600`。
4. 比较本地与服务器端脚本和 manifest 的 SHA256。
5. 审计目录不得包含任何访问令牌、渠道 Key、数据库密码或上游 Key。

### 7. 应用和回滚

1. 再次确认活动槽、目标渠道和容器健康。
2. 仅在用户已授权这一精确模型集合后运行：

```bash
python3 skills/sync-aivrae-models/scripts/apply_manifest.py \
  path/to/manifest.json \
  --execute \
  --confirm-channel-id CHANNEL_ID
```

3. 应用顺序为：
   - 写入新增模型价格；
   - 创建新增模型元数据；
   - 原子式更新渠道模型列表并刷新 abilities；
   - 删除移除模型的价格键；
   - 软删除移除模型元数据；
   - 验证后台和公开价格。
4. 任一步失败时恢复原渠道、原 options 和原元数据，并删除本次创建的元数据。
5. 回滚有错误时立即报告具体残留，不继续执行更多写入。

### 8. 验证生产结果

1. 验证目标渠道仍启用，模型有序列表与目标一致。
2. 验证新增 ability 已启用、移除 ability 不存在。
3. 验证新增模型元数据活跃，移除元数据对公开 API 不可见；数据库软删除记录可保留审计。
4. 匿名读取 `/api/pricing`，核对新增模型、移除模型和每个价格字段。
5. 运行渠道真实请求：

```bash
python3 skills/sync-aivrae-models/scripts/verify_routes.py \
  path/to/manifest.json \
  --execute \
  --confirm-channel-id CHANNEL_ID
```

6. 对图像、音频、视频和 Embedding 响应执行内容级检查；管理渠道测试成功只证明路由、转换和上游响应，不等于媒体内容或向量正确。
   `verify_routes.py` 会将此类模型列入 `content_checks_required`，不会把内容标为已验证。
7. 验证活动应用、PostgreSQL 和 Redis 容器健康，确认临时认证已清理。
8. 将验证 JSON、错误日志和最终哈希写入审计目录。

## 完成标准

- 用户选择的精确新增/移除集合已实施，没有额外模型。
- 价格使用目标 Key 与模型共同可用的最高有效云雾分组，并可从快照重算。
- manifest 通过本地校验和 API dry-run，且不含秘密。
- 生产备份可解压并有 SHA256。
- 配置、metadata、abilities、公开价格和真实路由均已验证。
- 媒体模型和 Embedding 已做内容级验证；容量错误被明确区分为上游问题。
- 临时 root token 或其他临时凭据已清除并独立验证。
- 最终报告列出变更模型、价格、实测结果、备份路径、审计路径和残留风险。
