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
| `KEEP_SERVER_ON_FAILURE` | 失败后保留服务器用于调试 | `false` |
| `SERVER_STATE_PATH` | 服务器状态文件路径 | `.hetzner-server-state.json` |

## 使用示例

```bash
export HETZNER_TOKEN=...
export BUILD_SOURCE_DIR=/path/to/docker-lineage-cicd

go run ./cmd/lineage-builder
```

## 失败后保留服务器

当构建过程出现问题时，可以通过设置 `KEEP_SERVER_ON_FAILURE=true` 来保留服务器实例，以便登录调试。启用后，如果任何环节出错，服务器将不会被销毁，并会输出服务器的详细信息：

```bash
export KEEP_SERVER_ON_FAILURE=true
go run ./cmd/lineage-builder
```

失败时会输出类似以下信息：

```
⚠️  WARNING: Server is being kept alive due to KEEP_SERVER_ON_FAILURE=true
⚠️  Server details:
⚠️    ID: 12345678
⚠️    Name: lineageos-builder
⚠️    IP: 1.2.3.4
⚠️    SSH Port: 22
⚠️    Datacenter: fsn1-dc14
⚠️  To connect: ssh root@1.2.3.4 -p 22
⚠️  Server state saved to: .hetzner-server-state.json
⚠️  To cleanup later, run: lineage-builder --cleanup
```

## SSH 密钥注入

在 GitHub Actions 环境下运行时，工具会自动获取触发 workflow 的用户的 GitHub SSH 公钥（如果有），并注入到服务器中，方便用户在需要时通过 SSH 连接服务器进行调试。

## 清理残留资源

如果程序异常崩溃，服务器可能不会被释放而持续计费。工具会在创建服务器后将信息保存到状态文件（默认 `.hetzner-server-state.json`）。可以使用 `--cleanup` 参数来清理残留的服务器资源：

```bash
# 清理保存在状态文件中的服务器
go run ./cmd/lineage-builder --cleanup
```

清理命令会：
1. 读取状态文件中的服务器信息
2. 检查服务器是否仍然存在
3. 如果存在且可以删除，则删除服务器和 SSH 密钥
4. 清理状态文件

注意：
- 如果设置了 `KEEP_SERVER_ON_FAILURE=true`，清理命令不会执行
- 如果状态文件不存在或服务器已被删除，不会抛出异常
- 只有在服务器存在但删除失败的情况下才会抛出异常

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
    KEEP_SERVER_ON_FAILURE: false  # 设置为 true 可在失败时保留服务器
```

执行前请在仓库 Secrets 中设置必要的变量：

- `HETZNER_TOKEN`

### GitHub Actions 自动清理

action 配置了自动清理步骤，无论构建是否成功，都会尝试清理残留的服务器资源（当 `KEEP_SERVER_ON_FAILURE` 不为 `true` 时）。这确保了即使构建过程中出现异常，也不会持续计费。

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
