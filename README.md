# jms

`jms` 是一款轻量级、开源的 SSH 堡垒机。它通过统一的 SSH 入口提供多云资产发现、访问控制、操作审计、文件传输和批量任务，并支持人类使用的交互式 TUI 与 AI/自动化使用的非交互命令。

## 功能

- SSH TUI：选择目标服务器和 SSH 用户后进入终端。
- 身份认证：支持数据库用户、LDAP 和 SSH 公钥认证。
- 资产管理：支持 AWS、腾讯云及本地静态资产。
- 权限控制：支持按用户组、资产标签和上传/下载操作授权。
- 审批：支持内置审批及钉钉审批。
- 代理：可通过 SSH Proxy 访问隔离网络中的资产。
- 审计：记录终端会话、SSH 登录、文件传输和非交互命令。
- 文件传输：通过堡垒机执行单文件 SCP 上传和下载。
- 任务调度：支持批量 Shell 任务、定时执行和结果查询。
- Web 与 API：提供管理界面、Web 终端及 Swagger API 文档。
- 自动化：使用原生 SSH 公钥身份执行资产查询和远程命令，无需为 AI 单独签发 API 凭据。

## 组件

| 组件 | 默认端口 | 用途 |
| --- | ---: | --- |
| `jms sshd` | `22222` | SSH TUI、非交互命令和 SCP 入口 |
| `jms api` | `8013` | 管理 API、Swagger 和 WebSocket 终端后端 |
| `jms scheduler` | `6060` | 资产同步、任务调度和后台作业 |
| `jms-web` | `8080` | Web 管理界面和 Web 终端 |

## 快速开始

### Docker Compose

项目中的 [`docker-compose.yml`](docker-compose.yml) 可启动 PostgreSQL、SSHD、API、Scheduler 和 Web：

```bash
mkdir -p data/ssh data/audit

export JMS_CONFIG_PATH="$PWD/config.yaml"
export JMS_SSH_DIR="$PWD/data/ssh"
export JMS_AUDIT_DIR="$PWD/data/audit"

docker compose up -d --build
```

启动后可访问：

- SSH：`localhost:22222`
- Swagger：<http://localhost:8013/swagger/index.html>
- Web：<http://localhost:8080>

仓库中的 [`config.yaml`](config.yaml) 仅为示例配置。启用数据库功能时，请先把数据库地址等字段调整为实际环境；Compose 网络内的 PostgreSQL 主机名为 `pg`。

### 从源码运行

需要 Go 1.25 或更高版本：

```bash
go build -o jms .

./jms --config ./config.yaml sshd --port 22222
./jms --config ./config.yaml api --port 8013
./jms --config ./config.yaml scheduler
```

各子命令的完整参数可通过 `./jms <command> --help` 查看。

## 配置

主要配置项位于 [`config.yaml`](config.yaml)：

| 配置段 | 用途 |
| --- | --- |
| `profiles` | 云账号和区域，用于自动发现资产 |
| `keys` | 访问目标服务器使用的 SSH 用户、密钥或密码 |
| `proxies` | 跨网络访问目标资产的代理 |
| `withLdap` | LDAP 身份认证 |
| `withDB` | 数据库、权限、审计和管理功能 |
| `withAuth` | API JWT 签名和有效期 |
| `withDingTalk` | 钉钉审批及通知 |
| `withUpload` | 文件上传限制 |
| `withTerminal` | Web 终端设置 |

生产部署前至少应完成以下事项：

1. 替换所有示例账号、密码和 `jwtSecret`。
2. 限制 SSH、API、数据库和 Web 端口的网络访问范围。
3. 将配置文件、SSH 私钥和审计目录设置为仅服务账号可读写。
4. 不要把真实云凭据、密码、Token、私钥或内部地址提交到 Git。

## SSH 使用

### 交互式终端

```bash
ssh -p 22222 alice@bastion.example.com
```

连接后在两级菜单中选择目标服务器和 SSH 用户。启用数据库或 LDAP 认证后，可注册本地公钥以免密登录：

```bash
ssh-copy-id -p 22222 alice@bastion.example.com
```

### 非交互命令

非交互模式适合脚本和 AI Agent。它要求使用 SSH 公钥认证并启用数据库，沿用当前用户的资产权限、目标机密钥和审计策略。

```bash
# 查询有权访问的资产
ssh -p 22222 alice@bastion.example.com targets --query web --format json

# 查询目标资产可用的 SSH 用户；target 支持资产 ID、名称或 IP
ssh -p 22222 alice@bastion.example.com \
  users --target srv-123 --format json

# 执行普通命令；本地 SSH 退出码与目标命令退出码一致
ssh -p 22222 alice@bastion.example.com \
  run --target srv-123 --user deploy -- uname -a

# 管道、重定向等 Shell 语法必须显式使用 --shell
ssh -p 22222 alice@bastion.example.com \
  run --target srv-123 --user deploy --shell 'df -h | grep /data'

# 复杂脚本可用单行 Base64 传递，减少本地 Shell 转义歧义
script_b64="$(printf '%s' 'printf "%s\n" "hello world"' | base64 | tr -d '\n')"
ssh -p 22222 alice@bastion.example.com \
  run --target srv-123 --user deploy --shell-base64 "$script_b64"

# 同一 SSH 用户存在多把密钥时可精确指定
ssh -p 22222 alice@bastion.example.com \
  run --target srv-123 --user deploy --key example-key.pem -- id
```

仓库内置的 [jms-ssh Skill](.agents/skills/jms-ssh/SKILL.md) 描述了 AI Agent 的发现、执行、传输和故障处理流程。

### SCP 文件传输

SCP 路径格式为 `<目标用户>@<目标地址>[:或 #key_name=...:]<目标路径>`：

```bash
# 上传
scp -P 22222 ./artifact.tar.gz \
  alice@bastion.example.com:deploy@192.0.2.10:/tmp/artifact.tar.gz

# 下载
scp -P 22222 \
  alice@bastion.example.com:deploy@192.0.2.10:/tmp/result.log \
  ./result.log

# 目标资产未绑定密钥时显式选择 key_name
scp -P 22222 ./artifact.tar.gz \
  alice@bastion.example.com:deploy@192.0.2.10#key_name=example-key.pem:/tmp/artifact.tar.gz
```

当前 SCP 代理支持单文件上传和下载，不支持递归目录传输。

## API 与 Web

运行 `jms api` 后，可在 <http://localhost:8013/swagger/index.html> 查看并试用当前版本的 API。Web 前端默认通过 `jms-api` 访问管理接口和终端 WebSocket。

启用认证时，受保护的 API 需要 `Authorization: Bearer <token>`。Shell 任务鉴权和迁移说明见 [`docs/2026-03-13-shell-api-auth.md`](docs/2026-03-13-shell-api-auth.md)。

## Kubernetes

[`deployment.yaml`](deployment.yaml) 是示例清单。使用前应替换镜像、ConfigMap、Secret、存储和网络策略：

```bash
kubectl create namespace jms --dry-run=client -o yaml | kubectl apply -f -
kubectl apply -f deployment.yaml
```

不要把生产凭据直接写入清单或提交到仓库；应由集群 Secret 管理方案注入。

## 开发

```bash
# Go 编译、静态检查和测试
go build ./...
go vet ./...
go test ./...

# 前端检查和构建
cd web
npm ci
npm run lint
npm run build
```

常用容器目标可通过 `make help` 查看，例如 `make all`、`make sshd`、`make api` 和 `make web`。

功能变化请以 [提交历史](https://github.com/xops-infra/jms/commits/dev) 和 [Releases](https://github.com/xops-infra/jms/releases) 为准，避免 README 中的版本流水账过期。

## License

本项目基于 [`LICENSE`](LICENSE) 中的协议开源。

## 致谢

- [TNK-Studio/gortal](https://github.com/TNK-Studio/gortal)
