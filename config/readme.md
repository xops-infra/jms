配置文件样例`/opt/jms/.jms.yml`：

```yml
profiles:
  - name: "aws"
    ak: "AKxxx"
    sk: "xxx"
    regions:
      - "cn-northwest-1"
    cloud: aws

  - name: "tencent"
    ak: "AKxxx"
    sk: "xxx"
    regions:
      - "ap-shanghai"
    cloud: tencent

keys:
  keyName1: xxx-user.pem
  keyName2: xxx-user.pem

proxies:
  - name: "10-153"
    host: "121.x.x.x"
    port: 22
    ipPrefix: "10.153."
    sshUsers:
      sshUsername: root
      identityFile: xxx.pem
      password: 

withSSHCheck:
  enable: false
  alert:
    robotToken: "xxxx"
  ips:
    - "10.x.x.x"

withPolicy:
  enable: true
  pg:
    host: "pg"
    port: 5432
    username: "postgres"
    password: "postgres"
    database: "jms"

withLdap:
  enable: true
  host: "xxx"
  port: 389
  baseDN: "dc=corp,dc=xxx,dc=com"
  bindUser: "xxx"
  bindPassword: "xxx"
  userSearchFilter: "(sAMAccountName=%s)"
  attributes:
    - dn
    - sAMAccountName
    - email

withDingTalk:
  enable: true
  processCode: "PROC-xxx-xxx-xxxx-xxxx-xxxxx"
  appKey: "xxx"
  appSecret: "xxx"

```