# GitHub Actions 自动构建与部署

本项目新增了 `.github/workflows/deploy-image.yml`。推送到 `aivrae/main`
或 `aivrae` 分支后，GitHub Actions 会构建 Docker 镜像并推送到 GHCR，然后
通过 SSH 登录服务器，拉取新镜像并重启 `new-api` 服务。

## 服务器要求

- 已安装 Docker。
- 已安装 `docker compose` v2，或旧版 `docker-compose`。
- 部署目录里已有 compose 文件，默认文件名是 `docker-compose.yml`。
- compose 文件里后端服务名必须是 `new-api`。

服务器上的 compose 文件可以继续使用原项目的 `docker-compose.yml`。工作流会
临时写入一个 `docker-compose.image.override.yml`，只覆盖 `new-api` 服务的
镜像地址。重启命令使用 `up -d --no-deps --force-recreate new-api`，因此只替换
`new-api` 容器，不会重启或重建 PostgreSQL、Redis、端口、卷等配置。

## GitHub Secrets

在仓库的 `Settings -> Secrets and variables -> Actions -> Secrets` 中配置：

| 名称 | 必填 | 说明 |
| --- | --- | --- |
| `DEPLOY_HOST` | 是 | 服务器 IP 或域名；也兼容现有 `VPS_HOST` |
| `DEPLOY_USER` | 否 | SSH 用户；也兼容现有 `VPS_USER`，默认 `root` |
| `DEPLOY_SSH_PRIVATE_KEY` | 是 | 可登录服务器的私钥；也兼容现有 `VPS_SSH_KEY` |
| `DEPLOY_PATH` | 否 | 服务器上的部署目录，默认 `/opt/new-api-stack`；也兼容 `VPS_NEW_API_PATH` |
| `DEPLOY_PORT` | 否 | SSH 端口，默认 `22` |
| `DEPLOY_COMPOSE_FILE` | 否 | compose 文件名，默认 `docker-compose.yml` |
| `DEPLOY_SSH_KNOWN_HOSTS` | 否 | 固定服务器 host key；不填时工作流会用 `ssh-keyscan` 生成 |
| `DEPLOY_REGISTRY_USERNAME` | 否 | 拉取私有 GHCR 镜像时使用的用户名 |
| `DEPLOY_REGISTRY_TOKEN` | 否 | 拉取私有 GHCR 镜像时使用的 token，需要 `read:packages` |

如果 GHCR 镜像是私有的，必须配置 `DEPLOY_REGISTRY_USERNAME` 和
`DEPLOY_REGISTRY_TOKEN`，否则服务器无法拉取镜像。token 可以使用 GitHub PAT，
权限至少包含 `read:packages`。

## GitHub Variables

在 `Settings -> Secrets and variables -> Actions -> Variables` 中可选配置：

| 名称 | 默认值 | 说明 |
| --- | --- | --- |
| `DEPLOY_IMAGE_PLATFORMS` | `linux/amd64` | 构建平台，例如 `linux/amd64,linux/arm64` |
| `DEPLOY_PORT` | `22` | SSH 端口，也可以作为 Secret 配置 |
| `DEPLOY_COMPOSE_FILE` | `docker-compose.yml` | compose 文件名，也可以作为 Secret 配置 |

## 镜像标签

工作流会推送以下标签：

- `ghcr.io/<owner>/<repo>:aivrae-main`
- `ghcr.io/<owner>/<repo>:deploy-aivrae-main-<short-sha>`
- `ghcr.io/<owner>/<repo>:sha-<short-sha>`

服务器部署默认使用稳定分支标签，例如 `aivrae-main`。

## Aivrae 当前线上环境

当前 Aivrae 生产环境的 New API stack 位于：

```text
/opt/new-api-stack
```

compose 文件为：

```text
/opt/new-api-stack/docker-compose.yml
```

后端服务名是 `new-api`，与自动部署工作流匹配。工作流上线前会尝试执行：

```bash
/opt/new-api-stack/backup-db.sh
```

然后只覆盖 `new-api` 服务镜像并重启该服务，不会改动 PostgreSQL、Redis、卷挂载。
部署脚本会等待 `/api/status` 健康检查通过，并尝试删除不再使用的
`calciumion/new-api:latest` 旧镜像标签。

当前线上 Caddy 仍会把 `/docs`、政策页和静态资源交给 `/opt/aivrae-site`，
并把多条 New API 前端页面路由交给 `/opt/aivrae-site/newapi-overrides`。如果要
让新镜像内置的 New API 前端页面生效，需要同步调整 Caddy 路由或移除对应
`newapi-overrides` 覆盖规则。

目标切换状态下，Caddy 只需要反代到 `new-api:3000`，并保留必要的旧路径跳转：

```caddyfile
aivrae.com {
    encode gzip zstd

    @legacy_plans path /plans /plans.html
    redir @legacy_plans /wallet 302

    handle {
        reverse_proxy new-api:3000
    }
}
```

## 文档与公开页面配置

本仓库已经包含从旧静态站迁移过来的多语言文档与公开页面内容：

```text
content/documentation/docs
content/documentation/config.local.json
content/documentation/config.docker.json
```

- 配置仍按原项目的 Option 机制保存到数据库，key 为 `DocumentationSettings`。
- 本地开发：在“系统设置 -> 内容 -> Documentation”中粘贴
  `content/documentation/config.local.json`。
- Docker 部署：Markdown 内容会随镜像放到 `/app/documentation/docs`，在后台粘贴
  `content/documentation/config.docker.json` 后即可启用。
- 后端不会读取 `/data/docs/config.json`，也不会用文件配置覆盖数据库配置。

新镜像可以直接提供这些公开路由：

```text
/docs
/docs/<slug>
/terms
/privacy-policy
/refund-policy
/acceptable-use
/contact
```

因此正式切换时，Caddy 中这些路径不应再继续指向旧的 `/opt/aivrae-site` 静态站。
