# LineageOS-Hetzner-Build

该工具用于在 Hetzner Cloud 上启动临时实例，执行 docker-lineage-cicd 构建，并将生成的构建产物下载回本地。工具只负责远程构建与产物拉取，不再包含 Git Release 等后处理逻辑。

构建流程会在实例上检测并通过 get.docker.com 脚本安装 Docker 与 Docker Compose 插件（需使用具备 root 权限且包含 curl 的镜像），以确保 docker compose 可用。可选设置 `GET_DOCKER_SHA256` 用于校验安装脚本；未设置时会提示在生产环境中先审核脚本内容。安装依赖系统默认的 Docker 安装脚本与签名校验；如需自定义镜像或源，请确保 Docker 与 Compose 插件版本兼容且仓库可信。

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
| `BUILD_SOURCE_DIR` | 本地源目录（包含 docker-compose 与依赖文件） | 必填 |
| `BUILD_COMPOSE_FILE` | docker-compose 文件路径 | `docker-compose.yml` |
| `BUILD_WORKDIR` | 实例工作目录 | `lineageos-build` |
| `BUILD_TIMEOUT_MINUTES` | 构建超时时间（分钟） | `360` |
| `ARTIFACT_DIR` | 远程产物目录 | `zips` |
| `ARTIFACT_PATTERN` | 产物文件匹配 | `*.zip` |
| `LOCAL_ARTIFACT_DIR` | 本地保存产物目录 | `artifacts` |
| `KEEP_SERVER_ON_FAILURE` | 失败时保留服务器用于调试（true/false） | `false` |
| `USER_SSH_KEYS` | 用户 SSH 公钥（逗号分隔），自动注入到服务器 | (空) |
| `SERVER_STATE_FILE` | 服务器状态文件路径 | `.hetzner-server-state.json` |

## 使用示例

```bash
export HETZNER_TOKEN=...
export BUILD_SOURCE_DIR=/path/to/docker-lineage-cicd

go run ./cmd/lineage-builder
```

### 失败时保留服务器（调试用）

当构建失败需要登录服务器调试时，可以设置 `KEEP_SERVER_ON_FAILURE=true`：

```bash
export KEEP_SERVER_ON_FAILURE=true
export HETZNER_TOKEN=...
export BUILD_SOURCE_DIR=/path/to/docker-lineage-cicd

go run ./cmd/lineage-builder
```

如果构建失败，程序会输出服务器信息（ID、IP、SSH 连接命令等），但不会销毁服务器。你可以通过 SSH 连接到服务器进行调试。

**注意：** 服务器会持续计费直到手动销毁！

### 清理保留的服务器

调试完成后，使用 `--cleanup` 参数清理服务器：

```bash
# 确保 KEEP_SERVER_ON_FAILURE 未设置或为 false
go run ./cmd/lineage-builder --cleanup
```

程序会读取状态文件（默认 `.hetzner-server-state.json`），销毁服务器及相关 SSH 密钥。

### 添加 SSH 公钥用于远程访问

程序支持自动将 SSH 公钥注入到服务器，方便你直接 SSH 连接服务器：

1. **GitHub Actions 环境**：程序会自动从 GitHub 获取触发工作流的用户的 SSH 公钥（如有）
2. **本地环境**：程序会自动读取 `~/.ssh/id_*.pub` 文件
3. **手动指定**：通过 `USER_SSH_KEYS` 环境变量指定（逗号分隔）

```bash
export USER_SSH_KEYS="ssh-ed25519 AAAAC3... user@example.com,ssh-rsa AAAAB3... user2@example.com"
```

## GitHub Actions (Step 引用)

该仓库提供 composite action，可在其他仓库作为单个 step 引入。仓库会在本机（Actions Runner）打包源目录后再传输至 Hetzner。

```yaml
- name: LineageOS Build
  uses: Erope/LineageOS-Hetzner-Build@<tag-or-sha>
  with:
    HETZNER_TOKEN: ${{ secrets.HETZNER_TOKEN }}
    # 以下为可选
    HETZNER_SERVER_TYPE: cx41
    HETZNER_SERVER_LOCATION: fsn1
    HETZNER_SERVER_IMAGE: ubuntu-22.04
    HETZNER_SERVER_NAME: lineageos-builder
    HETZNER_SERVER_USER_DATA: ./cloud-init.yml
    HETZNER_SSH_PORT: 22
    BUILD_SOURCE_DIR: ./docker-lineage-cicd
    BUILD_COMPOSE_FILE: docker-compose.yml
    BUILD_WORKDIR: lineageos-build
    ARTIFACT_DIR: zips
    ARTIFACT_PATTERN: "*.zip"
```

执行前请在仓库 Secrets 中设置必要的变量：

- `HETZNER_TOKEN`

## 构建完成后的后处理建议

构建完成后，产物会存放在 `LOCAL_ARTIFACT_DIR` 目录。可以在后续步骤中自行处理，例如上传到对象存储或发布 Release。

### 使用 GitHub Actions 发布 Release 示例

```yaml
- name: Upload artifacts to release
  uses: softprops/action-gh-release@v2
  if: startsWith(github.ref, 'refs/tags/')
  with:
    files: artifacts/*.zip
  env:
    GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```
