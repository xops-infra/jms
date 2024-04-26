## 简介

`jms`是一款轻量级的云服务器链接工具，

- 支持 ldap 登录认证
- 支持多云服务器资产自动发现（目前支持 aws,tencent）
- 支持基于机器标签的权限管理
- 支持权限申请和审批功能（自带或者接入钉钉）
- 支持文件上传下载
- 支持机器可连接性监控告警功能
- 审计功能：
  - 支持操作日志回放功能，文本文件方式记录标准输入输出；
  - 支持文件上传下载行为入表 `record_scp`；
  - 支持服务器登录行为入表 `record_ssh_login`；

## 如何部署

云接入 JMS 准备工作：

1. 需要云上有一个服务器只读权限的服务用户，提供 AKSK，jms 将自动同步资产；
2. 启动方式支持 docker 和 k8s 部署，具体可以参考下面的使用手册；

## 特别感谢

- [TNK-Studio/gortal](https://github.com/TNK-Studio/gortal.git)

## 设计拓扑

![](.excalidraw.png)

## 使用手册

```bash
# 设置免密登录
# ssh-copy-id -p 22222 登录用户@jms域名
ssh-copy-id -p 22222 zhoushoujian@localhost

# 登录
# ssh -p 22222 登录用户@jms域名
ssh -p 22222 zhoushoujian@localhost

# 权限
# 基于机器标签 tag做了 2 个策略
# 1. 机器标签Owner=登录用户，可以看到
# 2. 机器标签Team=登录用户所在的Team，可以看到

# 文件传输
# 上传 scp -P 22222 本地文件  登录用户@jms域名:远端服务器用户@远端服务器IP地址:远端服务器文件路径
[root@zhoushoujianworkspace jms]# scp -P 22222 ./README.md  zhoushoujian@localhost:ec2-user@192.168.1.1:/tmp/README1.md
README.md                                     100% 2506     2.9KB/s   00:00
# 下载 scp -P 22222 登录用户@jms域名:远端服务器用户@远端服务器IP地址:远端服务器文件路径 本地文件
[root@zhoushoujianworkspace jms]# scp -P 22222 zhoushoujian@localhost:ec2-user@192.168.1.1:/tmp/README1.md /tmp/README.md
README1.md                                    100% 2506     1.8MB/s   00:00


# docker启动
docker run --rm --network=host -dit -v /root/jms/ssh/:/root/.ssh/ -v /root/jms/jms.yml:/opt/jms/.jms.yml -p 22222:22222 --name jms_test -e WITH_SSH_CHECK=true zhoushoujian/jms:latest

# k8s 部署，完善好 configmap配置后，直接部署即可
kubectl apply -f sstatefulset.yaml -n jms --create-namespace

```

## 大功能点设计思路讲解

### 1. 资产权限的申请和审批

申请方式支持 2 种：

1. 支持 cli 直接选择权限申请，快捷方便，但是功能单一；
2. 复杂策略的权限申请需要通过 API 调用的方式实现，具体可以查看 swagger 文档（http://localhost:8013/swagger/index.html），创建策略是通过 approval 审批接口申请；

如何通过审批？

1. 通过 API 调用的方式，可以通过 API 调用的方式实现；
2. 在使用系统默认工单审批方式时候，拥有 admin 组的用户还可以通过登录后的选择也查看和处理需要审批的工单；
   如果使用了外部关联的审批方案（即完全通过 API 实现权限管理），第二种就不会出现啦。

默认就有的策略：

1. tag:Owner=user;
2. tag:Team 和你 jms 用户信息组一致；

![服务日志](log.jpg)

## 开发计划

- [√] API 管理 Policy；
- [ ] 用户首次登录初始化用户信息，配置用户组；
- [√] 接入钉钉审批功能；
- [ ] 优化文件传输方式，支持 scp 后文件选择传输，简化传输命令；

## 开发日志

### 2024-04

- feat: 支持 API 方式管理 KEY 和云账号 Profile
- feat: 增加数据库 Record 表，记录上传下载和服务器登录日志
- feat: 支持数据库热加载配置，支持 API 操作 Key,Profile,Proxy；
- feat: 支持服务器按名称排序；
- feat: 支持密钥本地和数据库入库认证；

### 2024-01

- feat: 支持钉钉审批功能：
- feat: 支持 audit 日志定时清理
- feat: 支持服务器标签 EnvType !不等于的匹配规则

### 2023-12

- feat: 增加 API 管理；
- chore: 优化交互界面；
- feat:支持会话超时退出功能；
- feat: 支持基于 sqlite 的独立审批功能；

### 2023-11

- 支持监控机器连接性告警功能；
- 支持 scp 复制功能；
- 支持配置热更新；

### 2023-10

- 支持 ssh-copy-id 设置，并通过密钥验证登录；

### 2023-09

- 支持输入过滤功能；
- 支持设置策略，只能看到授权的资产；
- 增加录像功能；

### 2023-08

- 基本功能上线
- 增加资产分类，基于账号和区域
- 增加 ldap 认证功能
