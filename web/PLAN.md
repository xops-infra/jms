**Title**  
Web 终端与文件传输（Vite React + TS、tmux 可恢复、分块续传）实现计划

**Summary**  
- 新增登录、WebSocket 终端、分块文件传输 API，复用现有策略/代理/密钥逻辑。  
- 前端基于 Vite + React + TypeScript，提供登录、终端、文件上传/下载页面。  
- 终端支持 tmux 恢复会话，文件上传支持分块断点续传；多副本部署下无共享状态（元数据入库，分块存储走共享介质）。

**Backend Scope (Gin)**  
1) 登录补全  
- 实现 `POST /api/v1/login`：校验 DB（或默认 jms/jms when WithDB=false），返回 `{token, expires_at}`，TTL 24h，从配置读取 secret/ttl。  
- 新增 `requireUser()` 中间件，所有新接口/WS 使用。  

2) WebSocket 终端（tmux 恢复）  
- 路由：`GET /api/v1/terminal/ws`；Query：`host`(必)、`user`(可选)、`cols`、`rows`、`session_id`(可选，用于恢复)。Header：`Authorization: Bearer <token>`.  
- Handler：  
  - JWT → `DescribeUser` → 权限校验：`CheckPermission(host, Connect)` or system policy。  
  - 选择 SSHUser：`GetSSHUsersByHostLive(host)` → `NewSSHClient`（自动代理）。  
  - 建立 SSH session + Pty。若提供/生成 `session_id`：执行 `tmux new -A -s <session_id>`；重连时 `tmux attach -t <session_id>`。tmux 不存在或失败则回退普通 shell 并告警到日志。  
  - WS 消息协议：  
    - 前端→后端：`{"type":"input","data":"..."}`, `{"type":"resize","cols":80,"rows":24}`, `{"type":"ping"}`  
    - 后端→前端：`{"type":"data","data":"..."}`, `{"type":"exit","code":0}`, `{"type":"pong"}`  
  - 保活：WS ping/pong 每 20s；SSH keepalive。  
  - 审计：若 `withVideo.enable`，复用现有日志写入；追加登录/退出记录到 DB（如已有表可重用）。  

3) 文件传输（分块续传 + 代理复用）  
- 路由设计：  
  - `POST /api/v1/files/upload/init` body `{host,path,size,sha256?,chunk_size?}` → 返回 `{upload_id, chunk_size, expires_at}`。  
  - `PUT /api/v1/files/upload/chunk` query `upload_id`, `index`，body 二进制分块。  
  - `POST /api/v1/files/upload/complete` body `{upload_id, total_chunks, sha256?}` → 组装并通过 SFTP 写入目标 `path`，记录审计。  
  - `POST /api/v1/files/upload/abort` body `{upload_id}` 清理暂存。  
  - `GET /api/v1/files/download` query `host`, `path`, `range?` → SFTP 读取流式回传，支持 Range。  
- 权限：每个接口进入时调用 `CheckPermission(argsWithServer, Upload/Download)`；下载同理。  
- 分块存储策略（多副本友好）：  
  - 元数据存 DB（表：upload_sessions），含 host/path/size/sha256/chunk_size/status/expires_at。  
  - 分块数据存共享介质（推荐 S3/MinIO；备选 NFS/PVC）。配置项 `withUpload.store` 描述存储类型与桶/路径；默认 NFS/PVC 路径 `/opt/jms/upload_tmp`.  
  - 单 Pod 拼装：`complete` 时顺序读分块 → 合并到临时文件 → SFTP 直传目标机 → 删除分块。  
- 限制与安全：  
  - Max chunk size、Max file size 配置；限制目标路径（阻止覆盖系统文件），校验 sha256 可选。  
  - 传输缓冲使用 `io.CopyBuffer`(128–256KiB)；速率限制可选 `rate.Limiter` per request。  

4) 配置与依赖  
- go.mod 新增：`github.com/gorilla/websocket`, `github.com/pkg/sftp`.  
- 配置增加：`auth.jwtSecret`, `auth.jwtTTL`, `upload.store.type (s3|fs)`, `upload.store.fsPath`, `upload.store.s3.bucket/endpoint/ak/sk`, `upload.maxSize`, `upload.chunkSize`, `terminal.tmux.enable`。  

**Frontend Scope (Vite + React + TS)**  
- 依赖：`xterm`, `xterm-addon-fit`, `axios`, `zustand`/`redux`(任选简单状态), `react-router`, `@tanstack/react-query`(可选请求态)。  
- 路由：`/login`, `/terminal`, `/files`.  
- 状态：`token`, `userInfo`, `currentHost`, `sessionId`, `upload queue`.  
- API 层：`apiClient` 注入 Bearer；`wsFactory` 生成带 token 的 WS URL。  
- 终端组件：封装 xterm，处理 resize/ping，支持在重连时携带 `session_id` 继续 tmux。  
- 文件上传：分片读取 `Blob.slice` 按 `chunk_size`，`init -> chunk -> complete`，失败重试、断点记录 localStorage。  
- 下载：`fetch` + Range 可选，展示进度条。  
- UI：简单表单 + 表格/列表，主题定制（非默认字体/配色）。  

**Testing & Acceptance**  
- 单测：JWT 解析、登录逻辑、WS handler（协议与 tmux 回退）、分块上传元数据状态流转、权限拒绝路径。  
- 集成/E2E：  
  - 登录成功/失败。  
  - 终端：建立连接、输入输出、resize、生存 10min ping/pong、断线后 session_id 重连恢复上下文。  
  - 文件：上传 <100MB、>1GB 分块续传中断后续传、下载 Range、权限拒绝。  
  - 多副本：在 LB 轮询下分块上传（跨 Pod chunk 写入/complete 成功），WS sticky 生效。  
- 性能：并发 50 个终端/10 个 1GB 上传无崩溃；内存无泄漏。  

**Assumptions / Defaults**  
- 目标主机安装 tmux；若缺失则自动回退普通 shell 并提示。  
- 有共享存储可用（优先 S3/MinIO；否则提供单一 PVC 路径），否则分块续传限定在同一 Pod。  
- API 与前端同域/同 HTTPS，LB 已支持 `/api/v1/terminal/ws` sticky。  
- `withDB.enable=true` 以启用权限/审计；默认端口保持 8013。  

**Delivery Steps (sequence)**  
1) 后端：实现 JWT 登录 + `requireUser`；配置项落地。  
2) 后端：WS 终端（tmux 支持）路由与协议；集成现有 SSH/代理/审计。  
3) 后端：分块上传/下载接口 + 元数据表 + 存储抽象(S3/FS)。  
4) 前端：Vite scaffold（React+TS）+ auth/路由/主题；接入登录。  
5) 前端：TerminalView + WS 协议 + 重连 tmux。  
6) 前端：分块上传/下载页面 + 进度/续传。  
7) 联调与 E2E，验证 LB 多副本场景。  
8) 文档与部署（Dockerfile/helm 静态资源、配置示例）。
