# GitHub Actions 构建与蓝绿部署

生产发布采用 Docker Compose + Caddy 蓝绿切换。提交代码不再直接重建线上
`new-api` 容器：push 只运行检查并构建不可变镜像，生产部署必须手工触发并输入
完整的 `image@sha256:digest`。

## 工作流行为

`.github/workflows/deploy-image.yml` 包含三个任务：

1. Pull Request 和 push 到 `aivrae/main`：运行 Go 测试与默认前端类型检查。
2. push 到 `aivrae/main`：构建镜像并推送 SHA 标签到 GHCR，在工作流摘要中输出 digest。
3. `workflow_dispatch`：经过 `production` Environment 后，将指定 digest 部署到非活动槽。

以下内容变更不会触发生产镜像构建：

```text
data/blog-drafts/**
docs/**
仓库根目录的 Markdown 文件
```

运行时文档位于 `content/documentation/**`，不在忽略范围内。应用部署不会再自动
写入数据库 `DocumentationSettings`；内容发布需要单独执行，避免应用切换前修改
仍由旧版本读取的配置。

## 生产拓扑

生产目录：

```text
/opt/new-api-stack
```

应用服务：

```text
new-api-blue    NODE_TYPE=slave，API 槽位
new-api-green   NODE_TYPE=slave，API 槽位
new-api-master  NODE_TYPE=master，不加入 Caddy upstream
```

三个服务共享现有 PostgreSQL、Redis、`SESSION_SECRET` 和 `CRYPTO_SECRET`，但使用
独立的日志与 `/data` 目录。API 槽位不映射宿主机端口，只通过 external network
`new-api-net` 供 Caddy 访问。

部署文件：

```text
/opt/new-api-stack/docker-compose.slots.yml
/opt/new-api-stack/deploy-blue-green.sh
/opt/new-api-stack/slots.env
/opt/new-api-stack/active-slot
```

`active-slot` 只能包含 `blue` 或 `green`。`slots.env` 保存每个槽位与主节点当前使用
的不可变镜像引用，不保存数据库密码或 API Key。

## GitHub 配置

### Secrets

| 名称 | 必填 | 说明 |
| --- | --- | --- |
| `DEPLOY_HOST` | 是 | 生产服务器 IP 或域名 |
| `DEPLOY_USER` | 否 | SSH 用户，默认 `root` |
| `DEPLOY_SSH_PRIVATE_KEY` | 是 | SSH 私钥 |
| `DEPLOY_PATH` | 否 | 默认 `/opt/new-api-stack` |
| `DEPLOY_PORT` | 否 | SSH 端口，默认 `22` |
| `DEPLOY_SSH_KNOWN_HOSTS` | 否 | 固定服务器 host key |
| `DEPLOY_REGISTRY_USERNAME` | 私有镜像必填 | GHCR 用户名 |
| `DEPLOY_REGISTRY_TOKEN` | 私有镜像必填 | 至少具有 `read:packages` |
| `DEPLOY_SMOKE_TOKEN` | 推荐 | 低额度、受限模型的部署测试 Token |

### Variables

| 名称 | 默认值 | 说明 |
| --- | --- | --- |
| `DEPLOY_IMAGE_PLATFORMS` | `linux/amd64` | 构建平台 |
| `DEPLOY_PORT` | `22` | SSH 端口 |
| `DEPLOY_SMOKE_MODEL` | 空 | 非流式和流式真实中转测试模型 |

仓库需创建 `production` Environment。生产任务使用固定并发组且
`cancel-in-progress: false`；服务器脚本还会使用 `flock`，防止两个发布同时执行。

## 发布流程

1. 从 push 工作流摘要复制完整镜像 digest。
2. 打开 `Build image and blue-green deploy` 工作流。
3. 选择 `Run workflow`，输入 `ghcr.io/...@sha256:...`。
4. 工作流确认非活动槽无残留连接，然后只重建该槽。
5. 执行 `/api/status`、Caddy 内网访问和 Token 模型列表检查。
6. 配置了 `DEPLOY_SMOKE_MODEL` 时，再执行真实非流式和流式请求。
7. Caddy 候选配置通过校验后执行 reload，将新槽设为第一 upstream。
8. 原活动槽保持运行，不会被部署任务自动停止。

应用当前没有优雅关闭，因此脚本拒绝重建仍有已建立 HTTP 连接的非活动槽。

## Caddy 配置

Caddyfile 使用标记块，由部署脚本只替换该块：

```caddyfile
# BEGIN NEW_API_UPSTREAM
reverse_proxy new-api-green:3000 new-api-blue:3000 {
    lb_policy first
    health_uri /api/status
    health_interval 5s
    health_timeout 2s
    health_fails 2
    health_passes 2
    fail_duration 30s
    max_fails 1
}
# END NEW_API_UPSTREAM
```

不配置 POST 自动重试，避免已到达上游的调用被代理重放并产生重复计费。配置先在
`aivrae-caddy` 容器内验证，之后使用 `caddy reload` 热加载，不重启 Caddy。

## 回滚

部署脚本在每次切换前保留 Caddyfile 备份。若 reload 或公网状态检查失败，会恢复
上一份配置并再次 reload。数据库不会自动回滚，原活动槽也不会被自动删除。

手工查看状态：

```bash
cd /opt/new-api-stack
./deploy-blue-green.sh status
```

手工切换到已健康的槽位：

```bash
./deploy-blue-green.sh switch blue
./deploy-blue-green.sh switch green
```

## 旧端口

首次部署使用的公网 `16980` 仅用于阶段性测试。正式蓝绿槽位不映射宿主机 API
端口；旧单容器在连接排空并停止后，该端口随之关闭。公开 API 始终使用：

```text
https://aivrae.com
```
