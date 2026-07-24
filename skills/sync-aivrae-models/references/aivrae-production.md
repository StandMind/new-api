# Aivrae 生产同步

## 环境发现

当前已知生产主机别名为 `lingyun1`，公开站点为 `https://aivrae.com`，部署根目录通常为 `/opt/new-api-stack`。这些是发现起点，不是永久事实。

连接前使用 `wan-server-environment` skill 读取当前 SSH 配置。保持严格 host key 校验。每次重新读取：

```text
/opt/new-api-stack/active-slot
docker ps
docker compose labels
```

从 API 按渠道名称解析目标 ID；不要永久硬编码历史渠道 ID。

## 管理 API

管理请求使用：

```text
Authorization: Bearer <system access token>
New-Api-User: <matching user id>
Content-Type: application/json
```

常用接口：

| 方法 | 路径 | 用途 |
| --- | --- | --- |
| GET | `/api/option/` | 读取完整 options |
| PUT | `/api/option/` | 更新单个 option map |
| GET | `/api/channel/:id` | 读取渠道 |
| PUT | `/api/channel/` | 更新渠道并刷新 cache/abilities |
| GET | `/api/models/search` | 精确检查活跃元数据 |
| POST | `/api/models/` | 创建元数据并刷新价格 |
| DELETE | `/api/models/:id` | 软删除元数据并刷新价格 |
| GET | `/api/pricing?lang=en` | 匿名公开价格 |
| GET | `/api/channel/test/:id` | 指定渠道真实请求测试 |

`/api/option/` 需要 root。系统访问令牌有效但权限不足时会返回权限错误；不要把它误报为令牌失效。

优先使用用户提供且已授权的 root 管理令牌。确实需要临时 root token 时，必须满足：

1. 用户已明确授权本次生产变更；
2. 通过 SSH 读取到目标 root 用户当前 `access_token IS NULL`；
3. 不覆盖现有 token；
4. 随机 token 只存在于单个远端进程环境；
5. `EXIT/HUP/INT/TERM` trap 按 token 条件清理；
6. apply 结束后独立查询 `access_token IS NOT NULL` 为 0；
7. token 不出现在审计、shell history、命令输出或文件中。

任一条件不满足时停止并请求合适的 root 管理认证。

## PostgreSQL 备份

生产写入前创建独立备份，不调用会清理旧备份的轮转脚本。参考流程：

```bash
set -euo pipefail
umask 077
cd /opt/new-api-stack
set -a
source ./.env
set +a
stamp=$(date +%F_%H%M%S)
backup="/opt/new-api-stack/backups/newapi_${stamp}_before_model_sync.sql.gz"
docker exec -e PGPASSWORD="$POSTGRES_PASSWORD" new-api-postgres \
  pg_dump -U "$POSTGRES_USER" "$POSTGRES_DB" |
  gzip -9 > "$backup"
gzip -t "$backup"
sha256sum "$backup" > "${backup}.sha256"
test -s "$backup"
```

不要输出 `.env` 或密码。确认 `pipefail` 生效，避免 `pg_dump` 失败但 gzip 仍返回成功。

## 审计目录

在 `/opt/new-api-stack/preflight-reports/` 下创建本次唯一目录：

```text
aivrae-model-sync-YYYYMMDD_HHMMSS/
```

目录 `0700`，文件 `0600`。保存：

- 上游价格快照或其可信哈希；
- 选择配置；
- manifest；
- apply 和 verify 脚本；
- dry-run 结果；
- apply 结果与 stderr；
- 路由测试结果；
- 本地/服务器哈希；
- 备份路径和 SHA256。

不得保存任何 token 或 Key。

## 渠道更新注意事项

一般渠道更新接口拒绝 `status` 字段。更新 payload 还应移除：

```text
created_time
test_time
response_time
balance
balance_updated_time
used_quota
key
status
```

保留其他渠道字段和 `id`，只修改 `models`。空 `key` 依赖 GORM 非零更新来保留原 Key，但脚本仍不得主动发送或记录渠道 Key。

模型列表是有序逗号字符串。移除旧模型时保持其他模型原顺序，再按用户选择顺序追加新模型。

价格 option 和模型元数据是全站共享配置。移除模型前分页读取全部渠道，若除目标渠道外还有任何启用或禁用渠道引用该模型，必须停止。不能为了单一渠道移除而删除其他渠道仍在使用的全局价格或元数据。

## 写入与回滚

推荐顺序：

1. 写新增模型价格；
2. 创建新增 metadata；
3. 更新渠道模型列表；
4. 删除旧模型的全部 managed option 键；
5. 删除旧 metadata；
6. 验证。

每次 option 写入前重新读取当前映射，只合并本次新增/移除模型的键；检测到这些模型的值被并发修改时停止。失败回滚也只恢复本次模型键，保留其他模型在执行期间产生的独立变更。渠道更新与回滚只修改 `models`，保留当前其他字段。

回滚顺序：

1. 恢复原渠道；
2. 恢复已删除的旧 metadata；
3. 删除本次创建的新 metadata；
4. 恢复全部已修改 option map。

每一个后置 option 删除也必须加入回滚追踪。API 调用结果不确定时先读取当前状态，不盲目重试创建。

## 验证

至少执行：

1. 管理 API 渠道模型列表；
2. PostgreSQL `abilities`；
3. 活跃 `models` 元数据和软删除状态；
4. 匿名公开 `/api/pricing`；
5. 指定渠道 route test；
6. 最终用户协议层 smoke test；
7. 媒体内容级检查；
8. 活动应用、PostgreSQL、Redis 健康；
9. 临时认证清理。

渠道测试返回成功只说明路由、适配器和上游响应成功。图像测试还要确认图像字段存在、可解码、尺寸正确且不是错误占位；Embedding 要确认向量非空且维度合理。

上游容量错误与配置错误分开记录。只有 manifest 明确允许 `upstream_capacity` 时，路由验证脚本才把容量饱和作为已知残留风险；它仍不能称为模型可用。
