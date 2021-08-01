
一个简单的，基于smux(多路复用)的内网穿透工具

# 特性

* 支持同时代理/穿透/监听(差不多一个意思)多个端口
* 断网自动重连
* 可动态更新真实服务器ip, 配合脚本，避免真实服务器出口ip变化时，手动修改中转服务的麻烦

# 说明

## 几个概念

* 真实服务器: 真实提供服务的服务器, 比如: nginx等服务所在的服务器
* 中转服务器: 也可以为代理服务器，需要有公网ip，可被客户端访问到
* 客户端: 普通的客户端，比如浏览器，app等

## 大概流程

中转服务器上启动 public-server

真实服务器上启动 local-server, 启动后，会主动连接public-server, 同时支持断线重连

客户端连接中转服务器

至此，完成

# 使用方法

## 中转服务器, 运行public-server

```bash
./public-server -ports=80,8080,3363,6379,15672... -server=真实服务器的出口ip -listen=0.0.0.0:12345 -listenapi=0.0.0.0:12346 -auth=用作动态更新真实服务器ip的验证的key 
```

```
-ports: 中转服务器监听的端口，对应真实服务器上各服务监听的端口
		10080:80,18080:8080,3363,6379 其中，p:q格式为 将中转服务器端口p映射为真实服务器端口q
		解释: 中转服务器端口10080映射到真实服务器端口80
				中转服务器端口18080映射到真实服务器端口8080
				中转服务器3363映射到真实服务器端口3363
-server: 真实服务器的公网出口ip, 用作限制仅此ip能连接public-server
-listen: 监听的地址:端口，用作监听真实服务器的连接，基于此连接，建立一条多路复用的转发通道
-auth: 当需因为真实服务器ip变化，需要动态修改server时，此字段起作用，用作验证发起动态修改的客户端的合法性, 可为任意长度字符串
-listenapi: 提供动态修改api服务监听的地址:端口
```

ports 格式举例10080:80,3363,18080:8080

中转服务器|真实服务器
---		| ---
10080 	| 80
3363 	| 3363
18080 	| 8080

## 真实服务器，运行local-server

```bash
./local-server -server=public-server's-addr:12345
```

```
-server: "ip:port" ip:中转服务器的 port:public-server中listen的端口
```

## 动态更新真是服务器地址

public-server上暴露了api `/updatesrv?srv=real-ip` 可通过定时脚本，动态更新真实服务器的公网出口ip到中转服务器上

例如，使用curl
```bash
curl -H "Authorization: auth key set in public-server's param" -X GET http://{public-server's ip}:{listenapi's port}/updatesrv?srv={real_ip}
```
