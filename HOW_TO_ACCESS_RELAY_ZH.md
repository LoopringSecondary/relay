# Loopring Relay 中文接入文档
这篇文章是一个教程，教您如何接入Loopring Relay（以下简称Relay），向路印协议提交订单。

## 什么是Loopring Relay？
Relay是钱包与Loopring协议之间的中继，向上和钱包对接，向下和Miner对接，一方面接收钱包提交的订单并维护Loopring订单池，另一方面为钱包提供支撑性功能，方便钱包或者用户了解整个市场的状况，同时将订单广播到其他Relay和Miner，增加订单的传播。所有的Relay组成了全网的订单池和深度。

注意：这里提到的Relay，是我们Github项目Relay以Relay的模式启动后，提供的中心化服务，而Relay同样可以以Miner的身份启动，只处理订单的撮合。要使用或者自己实现Miner，可以参考我们Github主页的Relay项目。

## Relay提供了哪些接入方式？
HTTP-JSON-RPC2.0是目前Relay最直接的接入方式，我们完全符合JSONRPC2.0规范，详细的接口文档和数据示例，请参考：[Relay API Spec](#https://github.com/Loopring/relay/blob/master/JSONRPC.md)

[Loopring.js](#https://github.com/Loopring/loopring.js)是我们开发Javascript版本的sdk，封装了所有JSONRPC接口，提供方法调用级别的支持，帮助Javascript项目快速接入Relay。

## 建议的接入步骤
Loopring相关的所有文档，代码都是开源的，所以您可以自由的阅读和使用。如果您是第一次接触Loopring，我们建议您按照如下顺序接入Relay：

    1. 了解Loopring，请参考我们的文档中心: https://docs.loopring.org
    2. 了解Loopring协议，阅读我们的白皮书: https://github.com/Loopring/whitepaper
    3. 了解我们的智能合约实现: https://github.com/Loopring/protocol，很多关键的概念都在合约里定义，所以这一步对接入Relay很重要
    4. 阅读本文档, 了解接入需要知道的一些信息
    5. 更进一步，如果您希望自己实现Relay或者Miner，请参考我们的Relay源代码: https://github.com/Loopring/relay

## 版本号
目前Relay最新版本1.0.0-BETA

## 术语表

名词 | 解释
---|---
Order | 符合Loopring protocol格式的订单数据
Allowance | 代币授权额度，这里通常指的是用户授权给Loopring protocol的额度
Balance | 用户代币资产余额
Fill | Order撮合产生的OrderFill事件数据
Depth | 市场深度
Ticker | 24小时市场变化统计数据
Trend | 市场变化趋势信息，目前维度仅支持1Hr
RingMined | Order组成的环撮合的结果
Cutoff | 用户以地址为单位设置的订单全部失效时间点，cutoff时间点之前的订单，全部变为无效订单
PriceQuote | 各个币种的市价参考，目前支持BTC, CNY, USD
Protocol | Loopring合约地址，伴随着合约升级，地址是不同版本的
LrcFee | 设置撮合需要的LrcFee
BuyNoMoreThanAmountB | 设置是否允许最终成交的TokenB超过AmountB
MarginSplitPercentage | 分润比例
Owner | 用户钱包地址
Market | 市场，我们目前只支持WETH市场
OrderHash | 订单的签名
TokenS | Token To Sell 要出售的Token, 请参考支持的Token列表
TokenB | Token To Buy 要买入的Token，请参考支持的Token列表
contractVersion | 合约版本号，目前我们release版本只有v1.0
timestamp | 这里特指提交订单时的timestamp，为订单生效时间，一般情况下，是当前时间
ttl | 订单有效期，失效日期=(timestamp + ttl)
salt | 随机数，用来解决重复下单问题


## JSON-RPC 重要接口介绍

下面着重介绍部分比较重要的接口

### loopring_submitOrder 提交订单

#### 生成订单
提交Loopring订单，是Relay最复杂的接口，涉及字段较多，以及对字段的签名。各个字段说明，请参考术语表。签名过程代码示例，请参考[路印协议v1.0.0订单结构和数字签名](https://github.com/Loopring/loopring.js/wiki/路印协议v1.0.0订单结构和数字签名)，签名生成的v,r,s, 在提交订单时一并提交。

#### 校验
订单提交到Relay后，我们会做一系列校验，包括但不限于：

    1. 基本地址和代币校验
    2. Token和交易对是否支持
    3. 合约版本校验
    4. 地址白名单
    5. 验签
 
然后存储在Relay的中心化数据库中，同时广播到其他Relay和Miner，等待撮合引擎的撮合或者用户的下一步操作。

#### 余额和授权
用户可用资产和中心化交易所有所不同，在用户余额不足或者授权额度不够的情况下，Relay支持先下单，后充值/授权，但我们建议最好在余额和授权都足够的情况下，做下单操作，减少撮合失败概率。要获取用户资产目前余额和授权在Loopring的使用情况，请使用looprig_getEstimatedAllocatedAllowance接口。

#### 支持的市场
现阶段的Relay只支持WETH市场的，满足ERC20标准的Token的交易。由于我们只ERC20 Token之间的交易，所以在交易开始前，需要将您的ETH余额转换成WETH余额，并授权给我们的Loopring协议，同时，如果您的手续费是通过LRC支付的，那LRC也需要授权。

#### 订单有效性

我们订单是否有效，由一下几个因素共同决定：
    
    1. status：订单状态
    2. timestamp：生效时间
    3. ttl：有效期
    4. cutoff：所有订单失效时间点

所以下单时，请注意设置合理的timestamp和ttl，同时检查cutoff是否合理。

### loopring_getOrders 获取订单列表
订单提交后，可以调用获取订单接口，获得订单信息，该接口支持多个维度的查询字段，具体见API文档。

### loopring_getDepth 获取深度

和中心化的交易所一样，订单生效后，会提现在深度数据里，同时订单失效后，深度数据不会包含您提交的订单。

Loopring的撮合依赖以太坊网络块生成时间，深度数据会概率性出现卖价比买家价格低的情况，那是因为订单已经提交撮合，正在等待Transaction被打包，订单状态仍然处于待成交状态。

### 获取撮合信息
您可以在API列表中找到RingMined和Fill的查询接口，这2个接口分别提供了环路撮合的环路和环路上每个节点的成交信息

### 为什么没有取消订单的接口？

要取消订单，必须提交Transaction到以太坊网络，调用Loopring合约的cancelOrder方法，所以没有提供单独的接口。

所以我们不建议频繁的取消订单，可以通过设置ttl和timestamp的方式来控制订单的生命周期，这样Relay可以在不提交撮合和消耗gas的情况下，提前阻止失效订单的撮合。

## 支持的Token列表

请访问我们合约[支持的Token列表](https://github.com/Loopring/protocol/blob/master/doc/tokens.md)页面

## 测试环境
Relay集成测试的过程中，我们建立了自己的测试环境，但目前不能保证环境足够稳定，我们正在改进，不久将提供一套稳定的测试环境，以及测试接入文档，提高合作伙伴的接入和测试效率。

## SDK & DEMO
TBD

## 联系我们
如果您有任何问题，请通过以下方式联系我们：
- Github Issues : https://github.com/Loopring/relay/issues
- 邮件：TBD
- Rocket.io: TBD
