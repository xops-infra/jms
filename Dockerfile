FROM amd64/centos:7
LABEL maintainer="zhoushoujianwork@163.com"
COPY ./bin/jms-linux-amd64 /usr/bin/jms-go
COPY ./entrypoint.sh /root/entrypoint.sh
RUN rm -f /etc/yum.repos.d/CentOS-*&&curl http://mirrors.aliyun.com/repo/Centos-7.repo > /etc/yum.repos.d/CentOS-Base.repo&&yum clean all&&yum makecache
RUN curl -o /usr/bin/gops 'https://devops-public-1251949819.cos.ap-shanghai.myqcloud.com/public/bin/linux_64_gops' && chmod +x /usr/bin/gops
RUN chmod +x /usr/bin/jms-go
RUN chmod +x /root/entrypoint.sh
WORKDIR /root
EXPOSE 2222
ENTRYPOINT /root/entrypoint.sh
