# LineageOS-Hetzner-Build

该工具用于在 Hetzner Cloud 上启动临时实例，执行 docker-lineage-cicd 构建，并将生成的构建产物下载回本地。工具只负责远程构建与产物拉取，不再包含 Git Release 等后处理逻辑。

构建流程会在实例上检测并通过 apt-get 安装 Docker 与 Docker Compose 插件（需使用 Debian/Ubuntu 镜像并具备 root 权限），以确保 docker compose 可用。

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

## 使用示例

```bash
export HETZNER_TOKEN=... 
export BUILD_SOURCE_DIR=/path/to/docker-lineage-cicd

go run ./cmd/lineage-builder
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
