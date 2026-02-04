# LineageOS-Hetzner-Build

该工具用于在 Hetzner Cloud 上启动临时实例，执行 docker-lineage-cicd 构建，并将生成的镜像发布到 GitHub Release。

## 环境变量

| 变量 | 说明 | 默认值 |
| --- | --- | --- |
| `HETZNER_TOKEN` | Hetzner Cloud API Token | 必填 |
| `HETZNER_SERVER_TYPE` | Hetzner 实例类型 | `cx41` |
| `HETZNER_SERVER_LOCATION` | Hetzner 实例位置 | (空) |
| `HETZNER_SERVER_IMAGE` | Hetzner 实例镜像 | `ubuntu-22.04` |
| `HETZNER_SERVER_NAME` | 实例名称 | `lineageos-builder` |
| `HETZNER_SERVER_USER_DATA` | Cloud-init user-data 文件路径 | (空) |
| `HETZNER_SSH_PORT` | SSH 端口 | `22` |
| `BUILD_REPO_URL` | docker-lineage-cicd 仓库地址 | 必填 |
| `BUILD_REPO_REF` | 仓库分支或 tag | (空) |
| `BUILD_REPO_TOKEN` | 访问私有仓库的 Token（仅在本机拉取时使用） | (空) |
| `BUILD_COMPOSE_FILE` | docker-compose 文件路径 | `docker-compose.yml` |
| `BUILD_WORKDIR` | 实例工作目录 | `lineageos-build` |
| `BUILD_TIMEOUT_MINUTES` | 构建超时时间（分钟） | `360` |
| `ARTIFACT_DIR` | 远程产物目录 | `zips` |
| `ARTIFACT_PATTERN` | 产物文件匹配 | `*.zip` |
| `LOCAL_ARTIFACT_DIR` | 本地保存产物目录 | `artifacts` |
| `GITHUB_TOKEN` | GitHub Token | 必填 |

## 使用示例

```bash
export HETZNER_TOKEN=... 
export BUILD_REPO_URL=https://github.com/xxx/docker-lineage-cicd.git
export GITHUB_TOKEN=...

go run ./cmd/lineage-builder
```

## GitHub Actions

该程序可直接运行在 GitHub Actions 中，只需提供上述环境变量并确保 Actions 有权限访问 Hetzner 与 GitHub Token。仓库会在本机（Actions Runner）完成拉取并打包后再传输至 Hetzner。

仓库已内置 `.github/workflows/lineage-build.yml`，默认使用 `workflow_dispatch` 手动触发。执行前请在仓库 Secrets 中设置：

- `HETZNER_TOKEN`
- `BUILD_REPO_URL`
- `GITHUB_TOKEN`
- `BUILD_REPO_TOKEN`（可选，仅用于本机拉取私有仓库）
