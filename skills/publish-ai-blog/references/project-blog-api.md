# 项目博客 API 与发布约束

## 代码来源

发布前以当前仓库为准，必要时重新读取：

- `model/blog_post.go`：字段、slug、状态、标签、翻译替换和发布时间语义。
- `model/localized_text.go`：支持语言和语言归一化。
- `router/blog-router.go` 与 `controller/blog.go`：公开和管理员 API。
- `web/default/src/features/blog/types.ts`：前端 payload 类型。
- `service/bloghtml/page.go`：服务端 SEO Markdown 渲染能力。

当前博客存储在数据库，不使用静态 Markdown 文件。管理员更新会删除该文章原有翻译后写入请求中的完整翻译集合，因此每次更新都必须发送所有要保留的语言。

## 当前语言与回退

当前支持：

```text
zh, en, es, fr, ru, ja, vi
```

公开读取优先使用请求语言，然后回退到 `en`，再回退到 `zh`，最后回退到任意非空翻译。这个回退只用于容错；本 skill 的正式发布要求七种语言都完整。

## Payload

```json
{
  "id": 0,
  "slug": "model-release-analysis",
  "status": "draft",
  "tags": ["ai", "llm", "api"],
  "cover_image": "https://cdn.example.com/blog/model-release.webp",
  "author": "Author Name",
  "published_time": 0,
  "translations": {
    "zh": {"title": "标题", "summary": "摘要", "content": "## 正文"},
    "en": {"title": "Title", "summary": "Summary", "content": "## Body"},
    "es": {"title": "Título", "summary": "Resumen", "content": "## Contenido"},
    "fr": {"title": "Titre", "summary": "Résumé", "content": "## Contenu"},
    "ru": {"title": "Заголовок", "summary": "Резюме", "content": "## Текст"},
    "ja": {"title": "タイトル", "summary": "要約", "content": "## 本文"},
    "vi": {"title": "Tiêu đề", "summary": "Tóm tắt", "content": "## Nội dung"}
  }
}
```

字段规则：

- `slug`：1 至 128 个字符，只允许 ASCII 小写字母、数字、点、下划线和连字符；首字符必须是字母或数字。
- `status`：只使用 `draft` 或 `published`。
- `tags`：最多 20 个，服务端会转为小写、去除 `#`、去重，并将单个标签截断到 32 个 Unicode 字符。skill 应在发送前主动满足这些规则。
- `cover_image`：可为空；正式文章优先使用稳定的公开 HTTPS URL。
- `author`：可为空，数据库字段上限为 128 个字符。
- `published_time`：Unix 秒。发布状态且为 `0` 时，服务端写入当前时间；未来时间表示定时公开。
- `translations`：键必须是受支持语言；每个正式版本都填写非空 `title`、`summary` 和 `content`。

## API

公开接口：

```text
GET  /api/blog/posts?lang=en&p=1&page_size=12&keyword=&tag=
GET  /api/blog/posts/{slug}?lang=en
POST /api/blog/posts/{slug}/view
```

管理员接口：

```text
GET    /api/blog/admin/posts?p=1&page_size=20&keyword=&status=
GET    /api/blog/admin/posts/{id}
GET    /api/blog/admin/posts/{id}/stats
POST   /api/blog/admin/posts
PUT    /api/blog/admin/posts/{id}
DELETE /api/blog/admin/posts/{id}
```

本 skill 的脚本故意不提供删除命令。

## 管理员认证

管理员接口支持系统访问令牌。请求必须同时发送：

```text
Authorization: Bearer <system-access-token>
New-Api-User: <admin-user-id>
Content-Type: application/json
```

令牌对应用户必须具有管理员角色。令牌可在个人资料的安全设置中生成或轮换。

只通过环境变量提供凭据：

```text
NEW_API_BASE_URL
NEW_API_ADMIN_USER_ID
NEW_API_ADMIN_ACCESS_TOKEN
```

不要把令牌直接写在 `export ...='token'` 命令、脚本参数、JSON、Markdown、版本控制文件或聊天中。优先使用已有秘密管理器，或在当前交互式 shell 中静默读取：

```bash
read -r -s -p 'Admin access token: ' NEW_API_ADMIN_ACCESS_TOKEN
export NEW_API_ADMIN_ACCESS_TOKEN
```

`NEW_API_BASE_URL` 填站点源地址，不包含尾部 `/api`。生产目标信息只从用户授权的环境配置或 `docs/project-local-notes.md` 读取，不复制进 skill。

## 脚本命令

所有命令从仓库根目录执行：

```bash
python3 skills/publish-ai-blog/scripts/blog_api.py validate post.json
python3 skills/publish-ai-blog/scripts/blog_api.py preflight
python3 skills/publish-ai-blog/scripts/blog_api.py admin-list --keyword model-release
python3 skills/publish-ai-blog/scripts/blog_api.py get-admin 42
```

远端写操作默认 dry-run：

```bash
python3 skills/publish-ai-blog/scripts/blog_api.py create-draft post.json
python3 skills/publish-ai-blog/scripts/blog_api.py create-draft post.json --execute

python3 skills/publish-ai-blog/scripts/blog_api.py update-draft 42 post.json
python3 skills/publish-ai-blog/scripts/blog_api.py update-draft 42 post.json --execute

python3 skills/publish-ai-blog/scripts/blog_api.py publish 42 post.json
python3 skills/publish-ai-blog/scripts/blog_api.py publish 42 post.json --execute --confirm-publish
```

如果 `update-draft` 会把已发布文章改回草稿，还必须添加 `--confirm-unpublish`。
`publish` 的 ID 必须来自真实的远端草稿或既有文章。纯本地准备流程不要编造 ID；停在 `create-draft` dry-run。用户说“不要发布到线上”或“只准备草稿”时，默认不执行任何带 `--execute` 的命令，包括远端草稿创建。

发布后验证：

```bash
python3 skills/publish-ai-blog/scripts/blog_api.py verify model-release-analysis --payload post.json
```

脚本支持全局 `--base-url` 和 `--user-id` 覆盖非秘密环境变量；访问令牌只从环境变量读取。

## 写入安全与重试

- `POST /api/blog/admin/posts` 不是幂等操作。创建超时后先在管理员列表中精确查找 slug，再决定是否重试。
- slug 全局唯一。脚本执行创建前会搜索同名文章，并在发现时拒绝创建。
- `PUT` 会整体替换翻译集合。不要发送部分语言更新。
- 管理 API 可能使用 HTTP 200 返回 `success: false`。脚本同时检查 HTTP 状态和业务状态。
- 发布结果不明时先 `get-admin`，不要盲目重复写请求。
- 不使用直接数据库写入绕开校验、事务和缓存行为。

## Markdown 的共同子集

React 页面支持较丰富的 Markdown，但服务端 SEO HTML 使用较小的解析器。为了保持首屏、爬虫和客户端一致，只使用：

```text
## 至 ###### 标题
段落
- 单层列表
1. 单层列表
> 引用
`行内代码`
围栏代码块
简单表格
[链接](https://example.com)
**粗体** 和 *斜体*
```

避免正文 H1、原始 HTML、脚注、嵌套列表、任务列表、Markdown 图片和自定义 directive。封面使用 `cover_image`。

## 线上验证

对每个语言调用：

```text
GET /api/blog/posts/{slug}?lang={locale}
```

同时打开：

```text
/blog/{slug}?lang={locale}
```

使用 `--payload` 时，脚本会比较公开返回的标题、摘要和正文是否与对应翻译一致。定时文章在 `published_time` 到达前不会出现在公开接口中。
