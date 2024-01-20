#!/bin/sh
mkdir -p /opt/logs/apps
# 判断是否有变量SSH_DIR 如果有则带上启动参数
if [ -n "$SSH_DIR" ]; then
    # 判断目录是否存在，不存在则创建
    SSH_DIR_FLAG="--ssh-dir $SSH_DIR"
    if [ ! -d "$SSH_DIR" ]; then
        mkdir -p "$SSH_DIR"
    fi 
fi

# 判断是否有变量DEBUG 如果有则带上启动参数
if [ -n "$DEBUG" ]; then
    DEBUG="--debug"
fi

/usr/bin/jms-go sshd $SSH_DIR_FLAG $DEBUG