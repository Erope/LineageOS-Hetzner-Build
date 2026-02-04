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

## GitHub Actions (可重用)

该仓库提供可重用的 GitHub Actions 工作流，方便其他仓库直接引用。仓库会在本机（Actions Runner）完成拉取并打包后再传输至 Hetzner。

在你的仓库中新增一个 workflow 文件，例如 `.github/workflows/lineage-build.yml`：

```yaml
name: LineageOS Build

on:
  workflow_dispatch:

jobs:
  build:
    uses: Erope/LineageOS-Hetzner-Build/.github/workflows/lineage-build-reusable.yml@main
    secrets:
      HETZNER_TOKEN: ${{ secrets.HETZNER_TOKEN }}
      BUILD_REPO_URL: ${{ secrets.BUILD_REPO_URL }}
      GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
      # 以下为可选
      HETZNER_SERVER_TYPE: ${{ secrets.HETZNER_SERVER_TYPE }}
      HETZNER_SERVER_LOCATION: ${{ secrets.HETZNER_SERVER_LOCATION }}
      HETZNER_SERVER_IMAGE: ${{ secrets.HETZNER_SERVER_IMAGE }}
      HETZNER_SERVER_NAME: ${{ secrets.HETZNER_SERVER_NAME }}
      HETZNER_SERVER_USER_DATA: ${{ secrets.HETZNER_SERVER_USER_DATA }}
      HETZNER_SSH_PORT: ${{ secrets.HETZNER_SSH_PORT }}
      BUILD_REPO_REF: ${{ secrets.BUILD_REPO_REF }}
      BUILD_REPO_TOKEN: ${{ secrets.BUILD_REPO_TOKEN }}
      BUILD_COMPOSE_FILE: ${{ secrets.BUILD_COMPOSE_FILE }}
      BUILD_WORKDIR: ${{ secrets.BUILD_WORKDIR }}
      BUILD_TIMEOUT_MINUTES: ${{ secrets.BUILD_TIMEOUT_MINUTES }}
      ARTIFACT_DIR: ${{ secrets.ARTIFACT_DIR }}
      ARTIFACT_PATTERN: ${{ secrets.ARTIFACT_PATTERN }}
      LOCAL_ARTIFACT_DIR: ${{ secrets.LOCAL_ARTIFACT_DIR }}
```

执行前请在仓库 Secrets 中设置必要的变量：

- `HETZNER_TOKEN`
- `BUILD_REPO_URL`
- `GITHUB_TOKEN`
