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
| `HETZNER_SSH_USER` | SSH 用户名 | `root` |
| `HETZNER_SSH_PORT` | SSH 端口 | `22` |
| `HETZNER_SSH_KEY` | SSH 私钥路径 | 必填 |
| `HETZNER_KNOWN_HOSTS` | known_hosts 文件路径 | 必填 |
| `BUILD_REPO_URL` | docker-lineage-cicd 仓库地址 | 必填 |
| `BUILD_REPO_REF` | 仓库分支或 tag | (空) |
| `BUILD_COMPOSE_FILE` | docker-compose 文件路径 | `docker-compose.yml` |
| `BUILD_WORKDIR` | 实例工作目录 | `lineageos-build` |
| `BUILD_TIMEOUT_MINUTES` | 构建超时时间（分钟） | `360` |
| `ARTIFACT_DIR` | 远程产物目录 | `out/target/product` |
| `ARTIFACT_PATTERN` | 产物文件匹配 | `*.zip` |
| `LOCAL_ARTIFACT_DIR` | 本地保存产物目录 | `artifacts` |
| `RELEASE_REPO_OWNER` | 发布仓库 owner | 必填 |
| `RELEASE_REPO_NAME` | 发布仓库 name | 必填 |
| `RELEASE_TAG` | Release tag | 必填 |
| `RELEASE_NAME` | Release 名称 | 默认等于 tag |
| `RELEASE_NOTES` | Release 描述 | (空) |
| `GITHUB_TOKEN` | GitHub Token | 必填 |

## 使用示例

```bash
export HETZNER_TOKEN=... 
export HETZNER_SSH_KEY=~/.ssh/id_rsa
export HETZNER_KNOWN_HOSTS=~/.ssh/known_hosts
export BUILD_REPO_URL=https://github.com/xxx/docker-lineage-cicd.git
export RELEASE_REPO_OWNER=xxx
export RELEASE_REPO_NAME=docker-lineage-cicd
export RELEASE_TAG=v1.0.0
export GITHUB_TOKEN=...

go run ./cmd/lineage-builder
```

## GitHub Actions

该程序可直接运行在 GitHub Actions 中，只需提供上述环境变量并确保 Actions 有权限访问 SSH 私钥与 GitHub Token。
