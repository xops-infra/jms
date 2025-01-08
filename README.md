## 1. 简介

`jms`是一款轻量级的云服务器链接工具，

- 登录认证方式
  1. 支持 ldap 登录认证；
  2. 数据库用户认证；
  3. 默认 jms/jms 用户认证；
- 支持多云服务器资产自动发现
  1. aws
  2. tencent
- 支持权限管理
  1. 基于用户组的权限管理；
  2. 基于机器标签的权限管理；
  3. [设计文档](https://www.yuque.com/motobox/enpuok/tzshwswnr7dhh6xp)
- 支持审批功能；
  1. jms 内置审批功能（普通用户 cli 发起，admin 用户 可以在 cli 审批）；
  2. 钉钉审批功能；
- 支持文件上传下载；
- 支持 Proxy 功能；
- 支持审计功能：
  1. 支持操作日志回放功能，文本文件方式记录标准输入输出；
  2. 支持文件上传下载行为入表 `record_scp`；
  3. 支持服务器登录行为入表 `record_ssh_login`；
- 支持服务器 SSH 可以连接性异常钉钉告警；
- 支持批脚本执行；
  1. 支持选定服务器；
  2. 支持定时任务反复执行；
  3. 支持接口任务执行结果，入参支持任务，或者某个服务器所有的执行历史；
  4. 执行状态，包括 "Pending", "Running", "Success", "Failed", "NotAllSuccess", "Cancelled"
- 支持设置全局通知功能；

## 2. 部署手册

- 准备工作：

  - （必须）云账号 AKSK（需要服务器查询权限）；
  - （必须）配置文件 `config.yml`，[配置介绍](config.yaml)；
  - （可选）ldap 认证账号；
  - （可选）钉钉审批；

- 启动 Server

  ```bash
  # 启动命令介绍
  $ jms sshd -h
  start sshd server as proxy server

  Usage:
    jms sshd [flags]

  Flags:
    -h, --help             help for sshd
        --log-dir string   log dir (default "/opt/jms/logs/")
        --port int         ssh port (default 22222)
        --timeout int      ssh timeout (default 1800)

  Global Flags:
    -c, --config string   config file (default is /opt/jms/config.yaml) (default "/opt/jms/config.yaml")
    -d, --debug           debug mode

  # 启动
  $ jms sshd --port 22222 --timeout 1800 --log-dir /opt/jms/logs/ --config ./config.yaml

  2024-04-28T20:52:59.706+0800    INFO    cmd/sshd.go:41  config file: /opt/jms/config.yaml
  2024-04-28T20:53:06.102+0800    INFO    cmd/sshd.go:74  enable policy
  2024-04-28T20:53:06.104+0800    INFO    instance/server.go:34   get instances profile: tencent-xxx region: ap-beijing
  2024-04-28T20:53:16.613+0800    INFO    cmd/sshd.go:114 starting ssh server on port 22222 timeout 1800...


  ```

- 启动 API 管理接口

  为了配合权限、用户、Key、云账号等信息的管理，提供了 API 管理接口，可以通过 API 方式管理 Key 和云账号 Profile。

  ```bash
  # 启动命令介绍
  $ jms api -h
  api server for jms, must withDB
          swagger url: http://localhost:8013/swagger/index.html

  Usage:
    jms api [flags]

  Flags:
    -h, --help             help for api
        --log-dir string   log dir (default "/opt/jms/logs/")
        --port int         api port (default 8013)

  Global Flags:
    -c, --config string   config file (default is /opt/jms/config.yaml) (default "/opt/jms/config.yaml")
    -d, --debug           debug mode

  # 启动后可以通过 http://localhost:8013/swagger/index.html 查看 API 文档
  ```

- 客户端连接和使用

  ```bash
  # 连接测试 默认config.yaml 没有使用ladp也没有使用数据库认证，默认用户密码 jms/jms
  $ ssh -p 22222 jms@localhost
  # 这里可以看到连接成功后的提示信息，且可连接的服务器数量为 0，因为没有配置云账号信息。

  # 配置免密登录，需要启用数据库或者 ladp 认证后才能实现
  # ssh-copy-id -p 22222 登录用户@jms域名
  $ ssh-copy-id -p 22222 zhoushoujian@localhost

  # 文件传输
  # 上传 scp -P 22222 本地文件  登录用户@jms域名:远端服务器用户@远端服务器IP地址:远端服务器文件路径
  $ scp -P 22222 ./README.md  zhoushoujian@localhost:ec2-user@192.168.1.1:/tmp/README1.md
  README.md                                     100% 2506     2.9KB/s   00:00
  # 下载 scp -P 22222 登录用户@jms域名:远端服务器用户@远端服务器IP地址:远端服务器文件路径 本地文件
  $ scp -P 22222 zhoushoujian@localhost:ec2-user@192.168.1.1:/tmp/README1.md /tmp/README.md
  README1.md                                    100% 2506     1.8MB/s   00:00

  ```

- 更多启动方式

  ```bash
  # docker启动
  $ docker run -dit -v ./config.yaml:/opt/jms/config.yaml -p 22222:22222 --name jms zhoushoujian/jms:latest

  # docker-compose 启动
  $ docker-compose up -d

  # k8s 部署，完善好 configmap配置后，直接部署即可
  $ kubectl apply -f statefulset.yaml -n jms --create-namespace
  ```

## 3. 开发计划

- v2 版本拆分组件支持分布式部署，拆分后的组件都是单机部署的支持容灾，sshd 多节点部署防止挂掉全部中断；
- 优化文件传输方式，支持 scp 后文件选择传输，简化传输命令；

## 4. 开发日志

- 2025-01

  - feat: 支持 scp 临时目录放到 app.App.Config.WithVideo.Dir 共用清理策略，否则还放 /tmp 由系统清理。

- 2024-12

  - feat: 支持人工修改服务资产登录用户密码而不是 KEY，并且刷新资产也能保留修改后的数据库配置

- 2024-11

  - refactor: 重构代码结构，拆分服务器入库，解耦 sdk 查询内存不释放问题；
  - feat:支持上传下载权限判断。

- 2024-09

  - feat: 支持使用 JMS_DINGTALK_WEB_HOOK_TOKEN 配置 runshell 任务发送钉钉消息；

- 2024-07

  - feat: 支持权限数据 Load 在内存，降低数据库 IO；
  - feat: 支持机器标签 KV 方式过滤而不是制定 team 和 envtype；
  - feat: 支持未被托管机器的可见但是报错，方便快速定位机器
  - feat: 审计增加实例 ID， 增加分钟级时间查询粒度，可作为准实时监控；

- 2024-06

  - feat: 支持本地配置链接机器
  - feat: 增加 aduit api 接口，支持查询审计日志；
  - feat: 增加连接数据库表同步到目标数据库表功能；

- 2024-05

  - refactor: 重构权限设计；
  - feat: 新增 shell task 功能，支持提交脚本任务执行，并支持查询任务执行结果；

- 2024-04

  - feat: 支持 API 方式管理 KEY 和云账号 Profile
  - feat: 增加数据库 Record 表，记录上传下载和服务器登录日志
  - feat: 支持数据库热加载配置，支持 API 操作 Key,Profile,Proxy；
  - feat: 支持服务器按名称排序；
  - feat: 支持密钥本地和数据库入库认证；

- 2024-01

  - feat: 支持钉钉审批功能：
  - feat: 支持 audit 日志定时清理
  - feat: 支持服务器标签 EnvType !不等于的匹配规则

- 2023-12

  - feat: 增加 API 管理；
  - chore: 优化交互界面；
  - feat:支持会话超时退出功能；
  - feat: 支持基于 sqlite 的独立审批功能；

- 2023-11

  - 支持监控机器连接性告警功能；
  - 支持 scp 复制功能；
  - 支持配置热更新；

- 2023-10

  - 支持 ssh-copy-id 设置，并通过密钥验证登录；

- 2023-09

  - 支持输入过滤功能；
  - 支持设置策略，只能看到授权的资产；
  - 增加录像功能；

- 2023-08

  - 基本功能上线
  - 增加资产分类，基于账号和区域
  - 增加 ldap 认证功能

## 5. 特别感谢

- [TNK-Studio/gortal](https://github.com/TNK-Studio/gortal.git)
