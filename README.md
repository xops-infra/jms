## 背景
1. 公司需要管理大量的资产，需要一个简单的工具来管理资产；
2. 每个团队都有连接资产的需求，需要一个简单的工具来连接资产；
3. 每个团队都有文件传输的需求，需要一个简单的工具来传输文件；
4. 介于公司还有大量的海外机器，还会出现访问速度慢的问题，Jumpserver在这块用来异步的 ansible来推送用户经常出现失败的痛点；

## 设计拓扑
![](.excalidraw.png)

## 功能

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

```

![服务日志](log.jpg)

## 开发日志
cli 部分参阅项目：https://github.com/TNK-Studio/gortal.git

### 2023-12
- 优化交互界面；
- feat:支持会话超时退出功能；
- feat: 支持基于 sqlite的独立审批功能；

### 2023-11
- 支持监控机器连接性告警功能；
- 支持scp复制功能；
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
- 增加 ldap认证功能
