# GitHub Actions 构建与蓝绿部署

生产发布采用 Docker Compose + Caddy 蓝绿切换。提交代码不再直接重建线上
`new-api` 容器：push 只运行检查并构建不可变镜像，生产部署必须手工触发并输入
完整的 `image@sha256:digest`。

## 工作流行为

`.github/workflows/deploy-image.yml` 包含三个任务：

1. Pull Request 和 push 到 `aivrae/main`：阻断执行全量 Go 测试、default typecheck、i18n 同步、default/classic production build、部署工具校验和容器镜像构建。
2. Pull Request 只验证镜像可构建，不推送；push 会推送 SHA 标签到 GHCR，并在工作流摘要中输出 digest。
3. `workflow_dispatch`：经过 `production` Environment 后，人工选择 `deploy`、`preflight-upgrade`、`start-upgrade`、`finalize-upgrade` 或 `rollback-upgrade`。

`.github/workflows/migration-compatibility.yml` 另外在 SQLite、MySQL 5.7、PostgreSQL
9.6 和 PostgreSQL 15 上运行两次 `--migrate-only`，并验证两项批准的列类型变化和
博客、系统任务、实例、权限表均存在。所有检查均为阻断式，不再使用
`continue-on-error`。

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

基础 Compose 与槽位 Compose 分工如下：

```text
docker-compose.yml        PostgreSQL、Redis
docker-compose.slots.yml  blue、green、master
```

完成首次迁移后，旧单容器服务必须从基础 Compose 中移除。若暂时保留停止容器用于
人工回滚，其 restart policy 必须设置为 `no`，防止宿主机重启后重新开放旧端口。

部署文件：

```text
/opt/new-api-stack/docker-compose.slots.yml
/opt/new-api-stack/deploy-blue-green.sh
/opt/new-api-stack/upgrade-preflight.sh
/opt/new-api-stack/observe-public.sh
/opt/new-api-stack/slots.env
/opt/new-api-stack/active-slot
/opt/new-api-stack/upgrade-state
/opt/new-api-stack/preflight-reports/
/opt/new-api-stack/deployment-history.log
```

`active-slot` 只能包含 `blue` 或 `green`。`slots.env` 保存每个槽位、主节点当前使用
的不可变镜像引用和服务级健康路径，不保存数据库密码或 API Key。`upgrade-state`
以原子替换方式记录候选镜像、旧 master 镜像、原活动槽、备份、Caddy 备份、持续
公网探测文件和当前阶段，权限为 `0600`。

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

`aivrae/main` 的分支保护应要求升级 PR 通过 `Validate application`、
`Build immutable image`、SQLite、MySQL 5.7、PostgreSQL 9.6 和 PostgreSQL 15
迁移检查后才能合并。工作流本身不使用 `continue-on-error`；是否真正禁止绕过合并
仍由 GitHub Required status checks 配置决定。

## 常规同架构发布

1. 从 push 工作流摘要复制完整镜像 digest。
2. 打开 `Build image and blue-green deploy` 工作流。
3. 选择 `Run workflow`，输入 `ghcr.io/...@sha256:...`。
4. 工作流确认非活动槽无残留连接，然后只重建该槽。
5. 执行 `/api/status`、Caddy 内网访问和 Token 模型列表检查。
6. 配置了 `DEPLOY_SMOKE_MODEL` 时，再执行真实非流式和流式请求。
7. Caddy 候选配置通过校验后执行 reload，将新槽设为第一 upstream。
8. 原活动槽保持运行，不会被部署任务自动停止。

应用收到 SIGTERM 后立即将 `/readyz` 置为 503，默认等待 15 秒让 Caddy 摘除节点，
再执行最长 900 秒的 `http.Server.Shutdown`。Compose 的 `stop_grace_period` 为
16 分钟。脚本仍拒绝重建存在已建立 HTTP 连接的非活动槽；真实中转冒烟最多重试
5 次，流式请求必须收到 SSE 数据和 `[DONE]`。

## 跨版本升级

升级必须按以下五次独立、人工批准的 workflow dispatch 执行：

1. `preflight-upgrade <digest>`：生成最新生产备份，在无宿主机端口的隔离 PostgreSQL
   15 + Redis 中恢复；旧 slave 持续读、数据库持续写时，候选镜像执行两次
   `--migrate-only`。随后依次验证候选 master/slave 和旧 master/slave，并拒绝包含
   未批准删除或既有对象变化的 catalog diff。原始 schema diff 和表、列、约束、
   索引、触发器、视图、序列快照一并写入 `preflight-reports/`。
2. `start-upgrade <digest>`：重新备份并校验，先启动独立的持续公网状态探测并在生产
   执行一次 `--migrate-only`；再停止旧 master、启动候选 master，确保任何时刻只有
   一个 master。随后再次确认非活动槽零连接，重建并执行 readiness、状态、模型、
   非流式和流式冒烟。长 SSE 必须通过 `aivrae.com` 建立，确认已经由旧活动槽返回
   首个数据帧且连接仍存活后才 reload Caddy。混合版本健康检查保持 `/api/status`。
3. `start-upgrade` 切流后持续观察 60 分钟。候选槽 unhealthy/restart，或公共状态与
   模型列表连续两次失败时，脚本立即按固定顺序回滚；全程公网探测出现非 2xx 也会
   阻断升级。成功后状态进入 `observing-complete`，旧槽仍保留原镜像。
4. `finalize-upgrade`：从切流时间起至少保留旧槽 24 小时，并确认旧槽连续 10 分钟
   零连接；随后才以候选 digest 重建旧槽作为 fallback，并把 Caddy 健康检查原子
   切换为 `/readyz`。
5. `rollback-upgrade`：仅允许在 finalize 前执行。先恢复 Caddy 到旧活动槽，再恢复
   旧 master；不会恢复数据库备份，也不会删除候选槽日志。

`upgrade-state` 的主要阶段为：

```text
preflight-complete -> starting -> switched -> observing-complete -> finalized
                                             \-> rolled-back
```

候选镜像必须是完整 `image@sha256:digest`。`start-upgrade` 还强制要求低额度
`DEPLOY_SMOKE_TOKEN` 和 `DEPLOY_SMOKE_MODEL`，否则不会进入生产迁移。

## Caddy 配置

Caddyfile 使用标记块，由部署脚本只替换该块：

```caddyfile
# BEGIN NEW_API_UPSTREAM
reverse_proxy new-api-green:3000 new-api-blue:3000 {
    lb_policy first
    health_uri /readyz
    health_headers {
        Connection close
    }
    health_interval 5s
    health_timeout 2s
    health_fails 2
    health_passes 2
    fail_duration 30s
    max_fails 1
}
# END NEW_API_UPSTREAM
```

两个槽都升级后使用 `/readyz`；首次混合版本阶段仍使用 `/api/status`。不配置 POST
自动重试，避免已到达上游的调用被代理重放并产生重复计费。配置先在
`aivrae-caddy` 容器内验证，之后使用 `caddy reload` 热加载，不重启 Caddy。
主动健康检查显式使用 `Connection: close`，避免 Caddy 的健康检查 keep-alive 被误判
为尚未完成的用户请求。部署脚本在同 upstream 顺序 reload 后最多等待 180 秒，让旧
transport 的空闲连接退出；真实长请求在此期间仍会保持连接并阻断槽位重建。

生产 Caddyfile 是只读单文件 bind mount。部署脚本会先把候选配置复制到容器
`/tmp/Caddyfile.candidate` 并执行校验，再用同目录临时文件和原子 `mv` 更新宿主机
Caddyfile，然后直接从容器内候选文件 reload，并通过 Caddy Admin API 确认目标槽和
回退槽已经进入运行时配置。这样既不会暴露部分写入状态，也不依赖容器内
`/etc/caddy/Caddyfile` 是否仍指向宿主机文件的最新 inode。

## 回滚

部署脚本在每次切换前保留 Caddyfile 备份。若 reload 或公网状态检查失败，会恢复
上一份配置并再次 reload。跨版本回滚固定先恢复 Caddy、再恢复旧 master。数据库
不会自动回滚，原活动槽和候选日志也不会被自动删除。
成功切换会追加写入 `deployment-history.log`，记录新活动槽、当前镜像、上一槽位和
上一镜像引用。

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

## 首次建槽观察（历史流程）

以下内容只用于从旧单容器首次迁移到蓝绿拓扑，不适用于跨版本升级。跨版本升级的
24 小时旧槽保留由 `finalize-upgrade` 强制校验，不允许用本节的人工批准缩短。

首次切入新槽后，默认建议从切流时刻开始观察至少 24 小时。可用独立 systemd
transient service 持续记录公共状态码和延迟：

```bash
systemd-run \
  --unit=aivrae-green-observation \
  --property=Type=exec \
  /opt/new-api-stack/observe-public.sh 86400 5
```

逐次结果保存在 `/opt/new-api-stack/observations/*.csv`，完成摘要保存在同名
`.summary` 文件。

缩短观察期不是默认流程，只能由生产负责人明确批准，并在本地运维记录中写明：

1. 实际观察时长、样本数、失败数和最大延迟。
2. Caddy、应用、PostgreSQL 和 Redis 的错误及重启统计。
3. 最新有效数据库备份。
4. 缩短观察期的批准时间和批准人。
5. 后续旧容器零连接排空结果。

没有明确批准时，观察期未满不得创建正式 blue/master、停止旧容器或关闭旧端口。

## 首次蓝绿迁移完成条件（历史流程）

首次从单容器迁移到正式蓝绿拓扑时，必须同时满足：

1. blue、green、master 使用同一个已验证的不可变 digest。
2. blue/green 均为 slave，master 为 master，且密钥、数据库、Redis 配置一致。
3. Caddy Admin API 中只有 blue/green 两个 upstream，顺序与 `active-slot` 一致。
4. 旧容器连续至少 10 分钟零连接，并覆盖至少一个数据刷新周期。
5. 停止旧容器后公共域名保持正常，旧宿主机端口不再监听。
6. 旧服务已从正式 Compose 移除，停止容器不会自动重启。
7. master 健康且不在 Caddy upstream 中。
8. 使用同 digest 完成 blue -> green 和 green -> blue 往返切换。
9. reload 前已建立的 SSE 请求能够完整收到 `[DONE]`。
10. Caddy、PostgreSQL 和 Redis 在整个迁移过程中没有重启。
