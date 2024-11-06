FROM amd64/centos:7
LABEL maintainer="zhoushoujianwork@163.com"
COPY ./bin/jms-linux-amd64 /usr/bin/jms-go
COPY ./entrypoint.sh /root/entrypoint.sh
RUN chmod +x /usr/bin/jms-go
RUN chmod +x /root/entrypoint.sh
WORKDIR /root
EXPOSE 2222 6060
ENTRYPOINT /root/entrypoint.sh
