![xfrps](https://github.com/liudf0716/xfrps/blob/master/logo.png)

## What is xfrps and why xfrps

xfrps was [xfrpc](https://github.com/liudf0716/xfrpc)'s server, it was a branch of [frp](https://github.com/fatedier/frp) which mainly focus on serving amount of routers and IOT devices 

The reason to start xfrps project is the following: 

1, we need a stable frp server to serve our [xfrpc](https://github.com/liudf0716/xfrpc), however frp change very fast, high version of frp doesn't promise compatible with low version 

2, frp's roadmap doesn't satisfy xfrp's need, for example, ftp support is important for us, but frp didn't support it

3, by maintaining our own frp server project, we can develope our own feature, no need to wait frp's support

## Different between xfrps and frp

xfrps start from v0.11.0 of frp, the following is difference

#### xfrps need client provide runid, if not, it will reject it

#### xfrps support only one tcp&ftp proxy for every client, and xfrps'client don't need to provide its remote port, xfrps will choose one for it

client can use its runid to get its tcp&ftp proxy's remote port by http request 

for example 
curl http://xfrps_domains:7500/api/port/tcp/getport/your_runid

#### xfrps support ftp

in order to use ftp proxy, u need add the following content to config file 

```[ftp]
type = ftp
local_ip = your ftp server ip
local_port = 21
```

to get ftp remote port, using the following http request:

curl http://xfrps_domains:7500/api/port/tcp/getftpport/your_runid


## How to contribute our project

See [CONTRIBUTING](https://github.com/liudf0716/xfrps/blob/master/CONTRIBUTING.md) for details on submitting patches and the contribution workflow.

## Contact

QQ群 ： [331230369](https://jq.qq.com/?_wv=1027&k=47QGEhL)


## Please support us and star our project

