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
- 支持审批功能；
  1. jms 内置审批功能（普通用户 cli 发起，admin 用户 可以在 cli 审批）；
  2. 钉钉审批功能；
- 支持文件上传下载；
- 支持 Proxy 功能；
- 告警功能；
  1. 支持服务器 SSH 可以连接性异常钉钉告警；
- 审计功能：
  1. 支持操作日志回放功能，文本文件方式记录标准输入输出；
  2. 支持文件上传下载行为入表 `record_scp`；
  3. 支持服务器登录行为入表 `record_ssh_login`；

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

优化文件传输方式，支持 scp 后文件选择传输，简化传输命令

## 4. 开发日志

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
