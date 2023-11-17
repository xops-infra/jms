## 背景
1. 公司需要管理大量的资产，需要一个简单的工具来管理资产；
2. 每个团队都有连接资产的需求，需要一个简单的工具来连接资产；
3. 每个团队都有文件传输的需求，需要一个简单的工具来传输文件；
4. 介于公司还有大量的海外机器，还会出现访问速度慢的问题，Jumpserver在这块用来异步的 ansible来推送用户经常出现失败的痛点；

## 设计拓扑
![](.excalidraw.png)

## 功能

### config
1. 支持配置个人信息，登录用户名密码，如果没有文件则登录 cli 等待输入用户名密码；
2. 支持配置多个账号，登录时可选择账号；

### 连接功能
1. 登录后，ls 查看可以连接的资产；
2. 支持模糊搜索，列表实时显示搜索结果；
3. 支持连接多个资产，连接后可通过 tab 切换；

### 文件传输功能
1. 支持文件上传下载；
2. 支持文件夹上传下载；

### 代理功能
1. 支持代理功能，可通过代理连接资产；
2. 支持代理的增删改查；

### 管理功能
1. 支持资产管理，增删改查；
2. 支持资产分组管理，增删改查；
3. 支持资产标签管理，增删改查；
4. 支持资产批量导入导出；
5. 支持资产批量执行命令；
6. 支持用户管理，增删改查；
7. 支持用户分组管理，增删改查；
8. 支持权限管理，增删改查；
9. 支持用户接入ldap；

```bash
# ssh-copy-id
ssh-copy-id -p 22222 zhoushoujian@localhost

# 登录
ssh -p 22222 zhoushoujian@localhost

# 文件传输
# 上传
[root@zhoushoujianworkspace jms]# scp -P 22222 ./README.md  zhoushoujian@localhost:ec2-user@192.168.1.1:/tmp/README1.md
README.md                                     100% 2506     2.9KB/s   00:00    
# 下载
[root@zhoushoujianworkspace jms]# scp -P 22222 zhoushoujian@localhost:ec2-user@192.168.1.1:/tmp/README1.md /tmp/README.md
README1.md                                    100% 2506     1.8MB/s   00:00


# docker启动
docker run --rm --network=host -dit -v /root/jms/ssh/:/root/.ssh/ -v /root/jms/jms.yml:/opt/jms/.jms.yml -p 22222:22222 --name jms_test -e WITH_SSH_CHECK=true zhoushoujian/jms:latest

```

![服务日志](log.jpg)

## 开发日志
cli 部分参阅项目：https://github.com/TNK-Studio/gortal.git
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
