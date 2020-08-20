# v2simple
该项目实现了一个简单版本的V2Ray。如果你对V2Ray等代理软件的运行机制很好奇，又没有时间研究其庞杂的实现，不妨关注此项目，核心代码不过100行，其余代码就是各种协议的具体实现。

目前实现了 [VMess协议]([https://www.v2fly.org/developer/protocols/vmess.html) 的客户端和服务端以及一个简洁的路由，只需简单配置即可无缝对接如下配置的V2Ray客户端或者服务端：
 * [国内直连](https://guide.v2fly.org/basics/routing/cndirect.html)
 * [TLS](https://guide.v2fly.org/advanced/tls.html) 
 * [TLS分流器](https://guide.v2fly.org/advanced/tcp_tls_shunt_proxy.html) ，内建了一个简单的TLS分流器，无需额外安装 [分流器](https://github.com/liberal-boy/tls-shunt-proxy)

以下V2Ray的功能**暂不支持**：
 * VMess协议的AEAD认证以及动态端口
 * 多入口多出口路由
 * Mux/mKCP/WebSocket/H2等协议
 * 流量统计以及用户管理
 * DNS服务


## 编译和使用

### 编译
要求 Go >= 1.14:
```shell script
git clone https://github.com/jarvisgally/v2simple
cd v2simple && go build -o bin
```

### 客户端模式
客户端模式在本地启动一个socks5代理，对流量进行加密和伪装，随后转发到Q外的服务器上：
```shell script
# 先将项目中的client.example.json复制到bin目录中，修改里面远程服务器的域名，并重命名为client.json
bin/v2simple -f client.json
```

### 服务端模式
服务器模式监听一个可以远程连接的接口，将流量还原为原始流量随后直接访问目标网站：
```shell script
# 先将项目中的server.example.json复制到bin目录中，修改里面域名证书和私钥地址，并重命名为server.json
bin/v2simple -f server.json
```

### 配置文件
客户端模式`client.example.json`：
```json
{
  "local": "socks5://127.0.0.1:1081",
  "route": "whitelist",
  "remote": "vmesss://a684455c-b14f-11ea-bf0d-42010aaa0003@<fix-me>:443?alterID=4"
}
```
服务端模式`server.example.json`：
```json
{
  "local": "vmesss://a684455c-b14f-11ea-bf0d-42010aaa0003@0.0.0.0:443?alterID=4&cert=<fix-me>&key=<fix-me>",
  "remote": "direct://"
}
```

其中`local`项标识本地服务的协议，`remote`项标识远端服务器的协议，目前支持的协议：
 * `socks5`：仅服务端，用于监听本地的socks5代理请求
 * `direct`：直连
 * `vmess`：支持客户端和服务端
 * `vmesss`：使用tls的vmess，同样支持客户端和服务端，服务端还需要额外指定域名的证书和私钥地址

事实上纯vmess的协议是也是基于TCP的，在Q看来是未知协议的TCP连接，此乃最大特征，因此不推荐使用，在例子中均使用了`vmesss`，伪装成常规的https流量；这需要预先注册一个域名并申请TLS证书，具体做法可以参考[这里](https://guide.v2fly.org/advanced/tls.html) 。

`route`项表示路由模式，目前支持如下模式：
 * `whitelist`：白名单模式，如果匹配，则直接访问，否则通过代理访问
 * `blacklist`：黑名单模式，如果匹配，则通过代理访问，否则直接访问
 * 空白：不做匹配，全部流量通过代理访问，见上述`server.json`

目前项目中仅包含`whitelist`，每一行是一个域名、IP或者CIDR，来自V2Ray的`geosite:cn`、`geoip:cn`和`geoip:private`。


## FQ原理和VMess协议

FQ技术则通过加密和伪装等方案突破网络审查，目前普遍采用的方式都是客户端-服务端的方式；在客户端也就是用户自己的机器上，在流量进入互联网之前，对流量进行加密和伪装，从而穿透Q的层层审查，然后在服务端还原为原始的流量访问目标网站；服务端必须是Q外的服务器。

[TODO：图]

由于用户的网络环境五花八门，而且国际出口拥堵的时候会对一般线路降权处理，因此不少FQ服务商会在国内增加一个BGP节点进行中转；如，长城宽带没有自己的国际出口，是向电信和联通租用的国际出口，会限制访问国外网站的速度和稳定性（我个人经验是有可能限制在2Mbps下），通过国内中转节点的话就能避免这个问题。

[TODO: 图]

现在也有不少号称国际专线的方案，亮点在于国内外节点是通过一条物理专线进行直连，完全不会被受到审查；专线原本是服务于跨国企业的，非常贵并且需要持有资质；成本在那，就必然有超卖的情况，而且企业专线和机场混在一起做，最大概率就是普通客户出事，导致企业客户一起连带不能用；是成本和风险都比较高的解决方案。

[TODO：图]

V2Ray灵活的配置使得其能够胜任上述的各种情况，比如多入口多出口路由就尤其适用于中转服务器，内建多种协议配合使用也能够满足各种用户需求和多样的网络环境，VMess协议是V2Ray的原创协议，主要解决了两个问题：
 * 用户的鉴权
 * 数据的加密


## 核心代码


## TODO


## Credits

感谢：

 * [你也能写个 Shadowsocks](https://github.com/gwuhaolin/blog/issues/12) ，当前项目参考了里面关于SOCKS5协议的实现；如果标题党一下，本文也可改为「你也能撸一个 V2Ray」吧
 * 


## 最后

因为学习Go语言和对FQ技术感兴趣的缘故，花了两三个星期完成此项目，深感Go语言的强大。