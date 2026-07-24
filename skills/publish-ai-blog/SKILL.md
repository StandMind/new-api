---
name: publish-ai-blog
description: Research, write, illustrate, translate, validate, and publish evidence-rich multilingual blog posts about current LLM and AI developments for this project's database-backed blog. Default to substantial standalone AI or model analysis without forcing project, API-gateway, routing, or billing relevance. Use when Codex needs to discover recent AI topics, verify official sources or public statements, draft long-form AI analysis or an explicitly requested brief, translate a post into every supported locale, prepare an original cover, create or update a blog payload, publish or schedule a post, or verify a live article. 中文触发包括撰写 AI 博文、最新大模型资讯、深度大模型分析、搜集 AI 资料、引用行业人物言论、多语言博客、生成博文封面、创建博客草稿、发布或定时发布博文。
---

# 发布 AI 多语言博文

## 核心约束

- 将“最新”视为时效性要求。开始研究时记录当前日期、检索时间范围和资料截止时间。
- 研究选题前读取 [references/research-sources.md](references/research-sources.md)。写作和翻译前读取 [references/editorial-policy.md](references/editorial-policy.md)。生成 payload 或调用线上接口前读取 [references/project-blog-api.md](references/project-blog-api.md)。
- 将本仓库视为博客发布载体，而不是文章主题。除非用户明确要求项目或网关视角，否则标题、摘要、正文和结论都不加入本项目、API 网关、供应商路由、计费、项目兼容性或接入状态。
- 用户未指定文章类型时，默认写证据充分的深度分析，不默认压缩成发布快讯。以技术背景、独立证据、方法限制、成本、安全、实际影响和未知项增加内容丰度，不用同义改写或重复结论凑篇幅。
- 将官方公告、技术文档、模型卡、论文、代码仓库和原始访谈作为主要证据。将新闻媒体、社交平台和社区讨论作为补充或线索，不把转述当作原始事实。
- 不编造引语、发布日期、参数、价格、跑分、可用区域或项目兼容性。无法验证的内容明确标注为未知、传闻或待确认，并避免写进标题与摘要。
- 不声称本项目已经支持某个模型、协议或供应商，除非已从当前仓库代码、公开配置或线上接口验证。
- 保持项目中与 new-api 和 QuantumNous 有关的名称、归属、品牌与元数据不变。
- 不把账号、令牌、Cookie、SSH 私钥或生产环境秘密写入文章、payload、skill 文件、命令参数或聊天内容。只通过环境变量读取博客 API 凭据。
- 默认先创建线上草稿。只有用户明确授权上线时才执行正式发布；未获得授权时停在经过校验的本地稿或远端草稿。
- 将“不要发布到线上”“暂时不要上线”“只准备草稿”默认解释为禁止所有远端写入，包括创建远端草稿。只有用户明确说“创建线上草稿”或同等意思时才执行 `create-draft --execute`。
- 不通过数据库或 SSH 直接写博客。正常发布只调用项目的管理员博客 API。

## 工作流

### 1. 确定选题与范围

1. 明确目标读者、文章类型、资料时间窗、主要写作语言和期望发布时间；用户未指定类型时使用深度分析。
2. 用户未指定主题时，从最近 7 天和最近 30 天分别收集候选事件，按“新鲜度、证据质量、对模型使用者与 AI 行业的影响、可写深度、长期价值”排序。
3. 检查已有文章，避免重复同一事件。可先调用公开列表；配置管理员凭据后优先运行 `admin-list` 搜索草稿与已发布文章。
4. 选择能回答“发生了什么、为什么重要、证据是什么、限制是什么、读者可以做什么”的主题。不要只改写厂商新闻稿。

### 2. 建立证据账本

1. 从 [assets/research-ledger.template.md](assets/research-ledger.template.md) 复制研究账本到本次工作目录。
2. 为每条重要事实记录事件日期、来源发布日期、原始 URL、来源类型、原文支持、适用条件和置信度。
3. 对模型发布至少收集官方公告、技术文档或模型卡，以及独立评测、复现或可靠媒体证据。默认深度稿应形成官方材料、独立证据和历史或竞品背景三个证据层；独立材料确实不足时明确说明，不用更多厂商转述伪装成交叉验证。
4. 对人物言论找到原始帖子、演讲视频、播客、访谈文字稿或公司正式文章；保留完整上下文和原始语言。
5. 对跑分、价格、上下文长度、速率限制和可用性记录测试版本、日期、地区、套餐和厂商自报属性。
6. 遇到来源冲突时保留冲突，不自行拼接出一个看似确定的结论。

### 3. 形成文章角度

- 区分“已经上线”“已宣布但未开放”“有限预览”“第三方声称”和“推测”。
- 解释变化对模型能力、实际使用、开发工作流、成本、延迟、多模态、上下文、工具调用、安全、研究或行业的影响。只有用户明确要求时才扩展到本项目或网关实现。
- 加入必要前情：前代模型、行业基线、相关论文、竞争产品和时间线。
- 给出限制、反例和仍待验证的问题。保持分析性语气，避免营销式最高级和无依据预测。

### 4. 撰写主稿

1. 将文章标题放在 payload 的 `title`，正文从 `##` 开始，不重复一级标题。
2. 编写独立摘要，直接说明事件、时间和意义，不使用“震撼”“颠覆”等空泛措辞。
3. 按 [references/editorial-policy.md](references/editorial-policy.md) 的默认深度、章节覆盖和证据密度完成主稿；除非用户明确要求简报，否则不能因为已经回答“发生了什么”就提前收尾。
4. 使用 [references/project-blog-api.md](references/project-blog-api.md) 中服务端和客户端都能稳定渲染的 Markdown 子集。
5. 在相关句子上使用内联 Markdown 链接，并在文末添加“资料来源”章节，列出最关键的原始与独立来源。
6. 对快速变化的信息写明“截至 YYYY-MM-DD”。对引用内容注明说话人、场合和日期。
7. 只摘录支撑论点所需的最短原文，优先转述并链接原始材料，避免大段复制受版权保护的内容。
8. 翻译前先做内容丰度复核：检查背景、技术变化、数据口径、独立证据、成本或现实影响、限制、安全、适用场景和未知项是否有材料可写；缺少关键部分时继续研究或明确报告证据缺口。

### 5. 完成全部语言

1. 以事实核验完成的主稿作为唯一语义源，再翻译为当前博客支持的全部语言。
2. 每次先从 `model/localized_text.go` 或 `web/default/src/i18n/languages.ts` 确认语言列表。当前支持 `zh`、`en`、`es`、`fr`、`ru`、`ja`、`vi`。
3. 保持标题、摘要、章节、数字、版本号、代码、引语归属、风险提示和来源链接语义一致。
4. 按目标语言重写句序和表达，不逐字翻译；保留模型名、API 字段、公司名和代码标识的官方写法。
5. 使用仓库中的 `docs/translation-glossary*.md` 统一已有术语。没有对应词汇表时，以厂商目标语言文档和该语言技术媒体的通用写法为准。
6. 不用一种语言的正文填充另一种语言。无法可靠完成某种语言时停在草稿并报告缺口。

### 6. 生成并校验 payload

1. 从 [assets/blog-post.template.json](assets/blog-post.template.json) 创建本次文章 JSON，不直接修改模板。
2. 使用 ASCII 小写 slug；使用短而稳定的主题标签；没有经过验证的长期公开 HTTPS URL 时保持 `cover_image` 为空。
3. 保持 `status` 为 `draft`，首次创建时保持 `id` 和 `published_time` 为 `0`。
4. 运行：

```bash
python3 skills/publish-ai-blog/scripts/blog_api.py validate path/to/post.json
```

5. 解决所有错误，并在发布前逐项审阅警告。特别检查缺少语言、占位符、链接不一致、疑似秘密和正文中的一级标题。

### 7. 处理封面与数据呈现

1. 用户要求封面，或获准发布且文章需要封面时，优先使用原创生成图或许可明确的素材。生成原创位图时调用可用的图像生成 skill，并把最终文件保存到本次草稿目录的 `assets/` 下。
2. 博文封面默认使用宽幅 16:9 构图，准确表达文章主题；避免可读文字、第三方标志、水印、仿冒产品界面、虚构人物言论和看似真实但没有来源的数据图。
3. 在研究账本记录图像来源或生成工具、日期、最终提示词或编辑意图、本地路径、尺寸、哈希、许可或来源说明。
4. 本地草稿阶段不上传图片。只有用户授权相应线上写入后，才把封面上传到项目控制的持久静态目录或既有媒体服务；确认匿名 HTTPS `200`、正确内容类型和文件一致性后，再写入 `cover_image` 并重新校验 payload。
5. 当前正文使用兼容 Markdown 子集，不插入 Markdown 图片。数据较多时优先使用简单表格并解释口径；只有仓库代码已验证支持正文图片且用户要求时，才添加图表图片，且不得用生成模型编造数据或标签。

### 8. 创建和复核线上草稿

1. 按 [references/project-blog-api.md](references/project-blog-api.md) 设置环境变量并运行只读预检：

```bash
python3 skills/publish-ai-blog/scripts/blog_api.py preflight
```

2. 先查看 dry-run：

```bash
python3 skills/publish-ai-blog/scripts/blog_api.py create-draft path/to/post.json
```

3. 只有用户已经明确授权创建远端草稿时才执行：

```bash
python3 skills/publish-ai-blog/scripts/blog_api.py create-draft path/to/post.json --execute
```

4. 记录返回的文章 ID。修改草稿时发送完整的七语言 payload，因为更新接口会整体替换翻译集合。
5. 创建请求超时或响应不明时，先用 `admin-list --keyword <slug>` 查重，不要直接重试创建。
6. 用户要求本地准备或禁止线上操作时，只运行校验与 dry-run，并报告因没有管理员凭据而跳过的预检或查重。

### 9. 发布或定时发布

1. 给用户展示最终标题、摘要、slug、语言完整性、重要事实、来源、封面、作者和发布时间。
2. 未明确授权上线时保持草稿。明确授权且已经存在真实远端草稿 ID 后，先运行发布 dry-run：

```bash
python3 skills/publish-ai-blog/scripts/blog_api.py publish POST_ID path/to/post.json
```

3. 确认无误后执行带双重确认的发布命令：

```bash
python3 skills/publish-ai-blog/scripts/blog_api.py publish POST_ID path/to/post.json --execute --confirm-publish
```

4. 立即发布时将 `published_time` 设为 `0`，由服务端写入当前时间。定时发布时填写未来的 Unix 秒级时间戳。
5. 没有真实远端草稿 ID 时不要编造 ID 运行发布 dry-run；本地流程停在 `create-draft` dry-run 即可。

### 10. 验证线上结果

1. 已到发布时间后逐语言核对公开 API 与页面：

```bash
python3 skills/publish-ai-blog/scripts/blog_api.py verify POST_SLUG --payload path/to/post.json
```

2. 检查 HTTP 成功、标题、摘要、正文、语言、封面、来源链接、移动端可读性和公开 URL。
3. 定时文章尚未到时间时，使用 `get-admin POST_ID` 验证状态和时间，不把公开 404 当作发布失败。
4. 发布调用结果不明时先读取后台文章状态，确认后再决定是否重试更新。

## 完成标准

- 资料账本包含可追溯的原始来源、日期和冲突说明。
- 默认文章是独立的大模型或 AI 深度分析；未经用户明确要求，不含本项目、网关、路由、计费或项目接入叙事。
- 主稿达到编辑规范中的深度目标，并提供背景、技术细节、独立证据、数据口径、实际影响、限制、安全或未知项，不是短新闻稿扩写或重复填充。
- 所有当前支持语言均有完整的标题、摘要和正文，且事实与链接保持一致。
- payload 通过脚本校验，未包含占位符、秘密或不受支持的 Markdown 结构。
- 使用封面时，已保存本地资产和来源记录；线上 URL 稳定、公开、可匿名访问并返回正确图片类型。
- 未经授权没有产生线上发布；获授权发布后已逐语言验证公开结果。
- 最终报告列出资料截止日期、payload 路径、草稿或文章 ID、状态、发布时间、公开 URL 和任何残留风险。
