#!/bin/sh

# 判断是否有变量DEBUG 如果有则带上启动参数
if [ -n "$DEBUG" ] && [ "$DEBUG" = "true" ]; then
    DEBUG="--debug"
fi

# 超时时间设置
if [ -n "$TIMEOUT" ]; then
    TIMEOUT="--timeout $TIMEOUT"
fi

if [ -n "$API" ] && [ "$API" = "true" ]; then
    /usr/bin/jms-go api $DEBUG
elif [ -n "$SCHEDULER" ] && [ "$SCHEDULER" = "true" ]; then
    /usr/bin/jms-go scheduler $DEBUG
else
    /usr/bin/jms-go sshd  $DEBUG $TIMEOUT
fi