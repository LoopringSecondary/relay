
# Relay API Spec

Loopring Relays are nodes that act as a bridge between Ethereum nodes and Loopring compatible wallets. A relay maintain global order-books for all trading pairs and is resposible for broadcasting orders selfishlessly to selected peer-to-peer networks. 

Wallets can host their own relay nodes to facility trading using Loopring, but can also take advantage of public relays provided by Loopring foundation or other third-parties. Order-book visulzation services or order browsers can also set up their own relay nodes to dispaly Loopring order-books to their users -- in such a senario, wallet-facing APIs can be disabled so the relay will run in a read-only mode. 

This document describes relay's public APIs (JSON_RPC and WebSocket), but doesn't articulate how order-books nor trading history are maintained.

This document contains the following sections:
- Endport
- JSON-RPC Methods


## Endport
```
JSON-RPC  : http://{hostname}:{port}/rpc
JSON-RPC(mainnet)  : https://relay1.loopring.io/rpc
```

## JSON-RPC Methods 

* The relay supports all Ethereum standard JSON-PRCs, please refer to [eth JSON-RPC](https://github.com/ethereum/wiki/wiki/JSON-RPC).
* [loopring_getBalance](#loopring_getbalance)
* [loopring_submitOrder](#loopring_submitorder)
* [loopring_getOrders](#loopring_getorders)
* [loopring_getDepth](#loopring_getdepth)
* [loopring_getTicker](#loopring_getticker)
* [loopring_getFills](#loopring_getfills)
* [loopring_getTrend](#loopring_gettrend)
* [loopring_getRingMined](#loopring_getringmined)
* [loopring_getCutoff](#loopring_getcutoff)
* [loopring_getPriceQuote](#loopring_getpricequote)
* [loopring_getEstimatedAllocatedAllowance](#loopring_getestimatedallocatedallowance)
* [loopring_getSupportedMarket](#loopring_getsupportedmarket)

## JSON RPC API Reference

***

#### loopring_getBalance

Get user's balance and token allowance info.

##### Parameters

- `owner` - The address, if is null, will query all orders.
- `contractVersion` - The loopring contract version you selected.

```js
params: {
  "owner" : "0x847983c3a34afa192cfee860698584c030f4c9db1",
  "contractVersion" : "v1.0"
}
```

##### Returns

`Account` - Account balance info object.

1. `contractVersion` - The loopring contract version you selected.
2. `tokens` - All token balance and allowance info array.

##### Example
```js
// Request
curl -X POST --data '{"jsonrpc":"2.0","method":"loopring_getOrderByHash","params":{see above},"id":64}'

// Result
{
  "id":64,
  "jsonrpc": "2.0",
  "result": {
    "contractVersion":"v1.0",
    "tokens": [
      {
          "token": "LRC",
          "balance": "510000000000000000000",
          "allowance": "21210000000000000000000"
      },
      {
          "token": "WETH",
          "balance": "12300000000000000000000",
          "allowance": "2121200000000000000000"
      }
    ]
  }
}
```

***

#### loopring_submitOrder

Submit an order. The order is submitted to relay as a JSON object, this JSON will be broadcasted into peer-to-peer network for off-chain order-book maintainance and ring-ming. Once mined, the ring will be serialized into a transaction and submitted to Ethereum blockchain.

##### Parameters

`JSON Object` - The order object(refer to [LoopringProtocol](https://github.com/Loopring/protocol/blob/master/contracts/LoopringProtocol.sol))
  - `protocol` - Loopring contract address
  - `owner` - user's wallet address
  - `tokenS` - Token to sell.
  - `tokenB` - Token to buy.
  - `amountS` - Maximum amount of tokenS to sell.
  - `amountB` - Minimum amount of tokenB to buy if all amountS sold.
  - `timestamp` - Indicating when this order is created.
  - `ttl` - How long, in seconds, will this order live.
  - `salt` - A random number to make this order's hash unique.
  - `lrcFee` - Max amount of LRC to pay for miner. The real amount to pay is proportional to fill amount.
  - `buyNoMoreThanAmountB` - If true, this order does not accept buying more than `amountB`.
  - `marginSplitPercentage` - The percentage of savings paid to miner.
  - `v` - ECDSA signature parameter v.
  - `r` - ECDSA signature parameter r.
  - `s` - ECDSA signature parameter s.

```js
params: {
  "protocol" : "0x847983c3a34afa192cfee860698584c030f4c9db1",
  "owner" : "0x847983c3a34afa192cfee860698584c030f4c9db1",
  "tokenS" : "Eth",
  "tokenB" : "Lrc",
  "amountS" : 100.3,
  "amountB" : 3838434,
  "timestamp" 1406014710,
  "ttl": 1200,
  "salt" : 3848348,
  "lrcFee" : 20,
  "buyNoMoreThanAmountB" : true,
  "marginSplitPercentage" : 50, // 0~100
  "v" : 112,
  "r" : "239dskjfsn23ck34323434md93jchek3",
  "s" : "dsfsdf234ccvcbdsfsdf23438cjdkldy",
}
```

##### Returns

`String` - The submit success info.

##### Example
```js
// Request
curl -X POST --data '{"jsonrpc":"2.0","method":"loopring_submitOrder","params":{see above},"id":64}'

// Result
{
  "id":64,
  "jsonrpc": "2.0",
  "result": "SUBMIT_SUCCESS"
}
```

***

#### loopring_getOrders

Get loopring order list.

##### Parameters

- `owner` - The address, if is null, will query all orders.
- `orderHash` - The order hash.
- `status` - order status enum string.(status collection is : ORDER_NEW, ORDER_PARTIAL, ORDER_FINISHED, ORDER_CANCEL, ORDER_CUTOFF)
- `contractVersion` - the loopring contract version you selected.
- `market` - The market of the order.(format is LRC-WETH)
- `pageIndex` - The page want to query, default is 1.
- `pageSize` - The size per page, default is 50.

```js
params: {
  "owner" : "0x847983c3a34afa192cfee860698584c030f4c9db1",
  "orderHash" : "0xf0b75ed18109403b88713cd7a1a8423352b9ed9260e39cb1ea0f423e2b6664f0",
  "status" : "ORDER_CANCEL",
  "contractVersion" : "v1.0",
  "market" : "coss-weth",
  "pageIndex" : 2,
  "pageSize" : 40
}
```

##### Returns

`PageResult of Order` - Order list with page info

1. `data` 
  - `orginalOrder` - The original order info when submitting.(refer to [LoopringProtocol](https://github.com/Loopring/protocol/blob/master/contracts/LoopringProtocol.sol))
  - `status` - The current order status.
  - `protocol` - loopring protocol address.
  - `dealtAmountS` - Dealt amount of token S.
  - `dealtAmountB` - Dealt amount of token B.

2. `total` - Total amount of orders.
3. `pageIndex` - Index of page.
4. `pageSize` - Amount per page.

##### Example
```js
// Request
curl -X POST --data '{"jsonrpc":"2.0","method":"loopring_getOrderByHash","params":{see above},"id":64}'

// Result
{
  "id":64,
  "jsonrpc": "2.0",
  "result": {
    "data" : [
      {
          "orginalOrder" : {
              "protocol" : "0x847983c3a34afa192cfee860698584c030f4c9db1",
              "owner" : "0x847983c3a34afa192cfee860698584c030f4c9db1",
              "tokenS" : "0x2956356cd2a2bf3202f771f50d3d14a367b48070",
              "tokenB" : "0xef68e7c694f40c8202821edf525de3782458639f",
              "amountS" : "0xde0b6b3a7640000",
              "amountB" : "0xde0b6b3a7640000",
              "timestamp" : 1506014710,
              "ttl": "0xd2f00",
              "salt" : "0xb3a6cc8cc77e88",
              "lrcFee" : "0x470de4df820000",
              "buyNoMoreThanAmountB" : true,
              "marginSplitPercentage" : 50, // 0~100
              "v" : "0x1c",
              "r" : "239dskjfsn23ck34323434md93jchek3",
              "s" : "dsfsdf234ccvcbdsfsdf23438cjdkldy"
          },
          "status" : "ORDER_CANCEL",
          "dealtAmountB" : "0x1a055690d9db80000",
          "dealtAmountS" : "0x1a055690d9db80000",
      }
    ]
    "total" : 12,
    "pageIndex" : 1,
    "pageSize" : 10
  }
}
```

***

#### loopring_getDepth

Get depth and accuracy by token pair

##### Parameters

1. `market` - The market pair.
2. `contractVersion` - The loopring protocol version.
3. `length` - The length of the depth data. default is 20.


```js
params: {
  "market" : "LRC-WETH",
  "contractVersion": "v1.0",
  "length" : 10 // defalut is 50
}
```

##### Returns

1. `depth` - The depth data.
2. `market` - The market pair.
3. `contractVersion` - The loopring protocol version.

##### Example
```js
// Request
curl -X POST --data '{"jsonrpc":"2.0","method":"loopring_getDepth","params":{see above},"id":64}'

// Result
{
  "id":64,
  "jsonrpc": "2.0",
  "result": {
    "depth" : {
      "buy" : [
        [200.1, 10.3], [199.8, 2], [198.3, 23]
      ],
      "sell" : [
        [205.1, 13], [211.8, 0.5], [321.3, 33]
      ]
    },
    "market" : "LRC-WETH",
    "contractVersion": "v1.0",
  }
}
```

***


#### loopring_getTicker

Get 24hr merged ticker info from loopring relay.

##### Parameters
1. `contractVersion` - The loopring protocol version.


```js
params: [
    "v1.0"
]
```

##### Returns

1. `high` - The 24hr highest price.
2. `low`  - The 24hr lowest price.
3. `last` - The newest dealt price.
4. `vol` - The 24hr exchange volume.
5. `amount` - The 24hr exchange amount.
5. `buy` - The highest buy price in the depth.
6. `sell` - The lowest sell price in the depth.
7. `change` - The 24hr change percent of price.

##### Example
```js
// Request
curl -X GET --data '{"jsonrpc":"2.0","method":"loopring_ticker","params":["v1.0"],"id":64}'

// Result
{
  "id":64,
  "jsonrpc": "2.0",
  "result": [{
    "high" : 30384.2,
    "low" : 19283.2,
    "last" : 28002.2,
    "vol" : 1038,
    "amount" : 1003839.32,
    "buy" : 122321,
    "sell" : 12388,
    "change" : "-50.12%"
  }]
}
```

***

#### loopring_getFills

Get order fill history. This history consists of OrderFilled events.

##### Parameters

1. `market` - The market of the order.(format is LRC-WETH)
2. `owner` - The address, if is null, will query all orders.
3. `contractVersion` - the loopring contract version you selected.
4. `orderHash` - The order hash.
5. `ringHash` - The order fill related ring's hash.
6. `pageIndex` - The page want to query, default is 1.
7. `pageSize` - The size per page, default is 50.

```js
params: {
  "market" : "LRC-WETH",
  "contractVersion" : "v1.0",
  "owner" : "0x8888f1f195afa192cfee860698584c030f4c9db1",
  "orderHash" : "0xee0b482d9b704070c970df1e69297392a8bb73f4ed91213ae5c1725d4d1923fd",
  "ringHash" : "0x2794f8e4d2940a2695c7ecc68e10e4f479b809601fa1d07f5b4ce03feec289d5",
  "pageIndex" : 1,
  "pageSize" : 20 // max size is 50.
}
```

##### Returns

`PAGE RESULT of OBJECT`
1. `ARRAY OF DATA` - The fills list.
  - `protocol` - The loopring contract address.
  - `owner` - The order owner address.
  - `ringIndex` - The index of the ring.
  - `createTime` - The timestamp of matching time.
  - `ringHash` - The hash of the matching ring.
  - `txHash` - The transaction hash.
  - `orderHash` - The order hash.
  - `orderHash` - The order hash.
  - `amountS` - The matched sell amount.
  - `amountB` - The matched buy amount.
  - `tokenS` - The matched sell token.
  - `tokenB` - The matched buy token.
  - `lrcFee` - The real amount of LRC to pay for miner.
  - `lrcReward` - The amount of LRC paid by miner to order owner in exchange for margin split.
  - `splitS` - The tokenS paid to miner.
  - `splitB` - The tokenB paid to miner.
2. `pageIndex`
3. `pageSize`
4. `total`

##### Example
```js
// Request
curl -X GET --data '{"jsonrpc":"2.0","method":"loopring_getFills","params":{see above},"id":64}'

// Result
{
  "id":64,
  "jsonrpc": "2.0",
  "result": {
    "data" : [
      {
          "protocol":"0x4c44d51CF0d35172fCe9d69e2beAC728de980E9D",
          "owner":"0x66727f5DE8Fbd651Dc375BB926B16545DeD71EC9",
          "ringIndex":100,
          "createTime":1512631182,
          "ringHash":"0x2794f8e4d2940a2695c7ecc68e10e4f479b809601fa1d07f5b4ce03feec289d5",
          "txHash":"0x2794f8e4d2940a2695c7ecc68e10e4f479b809601fa1d07f5b4ce03feec289d5",
          "orderHash":"0x2794f8e4d2940a2695c7ecc68e10e4f479b809601fa1d07f5b4ce03feec289d5",
          "amountS":"0xde0b6b3a7640000",
          "amountB":"0xde0b6b3a7640001",
          "tokenS":"WETH",
          "tokenB":"COSS",
          "lrcReward":"0xde0b6b3a7640000",
          "lrcFee":"0xde0b6b3a7640000",
          "splitS":"0xde0b6b3a7640000",
          "splitB":"0x0",
          "market":"LRC-WETH"
      }
    ],
    "pageIndex" : 1,
    "pageSize" : 20,
    "total" : 212
  }
}
```

***

#### loopring_getTrend

Get trend info per market.

##### Parameters

1. `market` - The market type.

```js
params: [
  "LRC-WETH"
]
```

##### Returns

`ARRAY of JSON OBJECT`
  - `market` - The market type.
  - `high` - The 24hr highest price.
  - `low`  - The 24hr lowest price.
  - `vol` - The 24hr exchange volume.
  - `amount` - The 24hr exchange amount.
  - `open` - The opening price.
  - `close` - The closing price.
  - `start` - The statistical cycle start time.
  - `end` - The statistical cycle end time.

##### Example
```js
// Request
curl -X GET --data '{"jsonrpc":"2.0","method":"loopring_getTrend","params":{see above},"id":64}'

// Result
{
  "id":64,
  "jsonrpc": "2.0",
  "result": {
    "data" : [
      {
        "market" : "LRC-WETH",
        "high" : 30384.2,
        "low" : 19283.2,
        "vol" : 1038,
        "amount" : 1003839.32,
        "open" : 122321.01,
        "close" : 12388.3,
        "start" : 1512646617,
        "end" : 1512726001
      }
    ]
  }
}
```

***

#### loopring_getRingMined

Get all mined rings.

##### Parameters

1. `ringHash` - The ring hash, if is null, will query all rings.
2. `contractVersion` - The loopring contract version.
3. `pageIndex` - The page want to query, default is 1.
4. `pageSize` - The size per page, default is 50.

```js
params: {
  "ringHash" : "0xb903239f8543d04b5dc1ba6579132b143087c68db1b2168786408fcbce568238",
  "contractVersion" : "v1.0"
  "pageIndex" : 1,
  "pageSize" : 20 // max size is 50.
}
```

##### Returns

1. `data` - The ring info.(refer to [Ring&RingMined](https://github.com/Loopring/protocol/blob/3bdc40c4f319e8fe70f58f82563db49579094b5c/contracts/LoopringProtocolImpl.sol#L109)
  - `ringHash` - The ring hash.
  - `tradeAmount` - The fills number int the ring.
  - `miner` - The miner that submit match orders.
  - `feeRecepient` - The fee recepient address.
  - `txHash` - The ring match transaction hash.
  - `blockNumber` - The number of the block which contains the transaction.
  - `totalLrcFee` - The total lrc fee.
  - `time` - The ring matched time.
2. `total` - Total amount of orders.
3. `pageIndex` - Index of page.
4. `pageSize` - Amount per page.

##### Example
```js
// Request
curl -X GET --data '{"jsonrpc":"2.0","method":"loopring_getCandleTicks","params":{see above},"id":64}'

// Result
{
  "id":64,
  "jsonrpc": "2.0",
  "result": {
     "data" : [
       {
        "ringhash" : "0xb903239f8543d04b5dc1ba6579132b143087c68db1b2168786408fcbce568238",
        "tradeAmount" : 3,
        "miner" : "0x8888f1f195afa192cfee860698584c030f4c9db1",
        "feeRecepient" : "0x8888f1f195afa192cfee860698584c030f4c9db1",
        "txHash" : "0x8888f1f195afa192cfee860698584c030f4c9db1",
        "blockNumber" : 10001,
        "totalLrcFee" : 101,
        "timestamp" : 1506114710,
       }
     ]
     "total" : 12,
     "pageIndex" : 1,
     "pageSize" : 10
  }
}
```
***

#### loopring_getCutoff

Get cut off time of the address.

##### Parameters

1. `address` - The ring hash, if is null, will query all rings.
2. `contractVersion` - contract version of loopring protocol.
3. `blockNumber` - "earliest", "latest" or "pending", default is "latest".

```js
params: [
  "0x8888f1f195afa192cfee860698584c030f4c9db1",
  "v1.0",
  "latest"
]
```

##### Returns
- `string` - the cutoff timestamp string.

##### Example
```js
// Request
curl -X GET --data '{"jsonrpc":"2.0","method":"loopring_getCutoff","params":{see above},"id":64}'

// Result
{
  "id":64,
  "jsonrpc": "2.0",
  "result": "1501232222"
```
***

#### loopring_getPriceQuote

Get the USD/CNY/BTC quoted price of tokens

##### Parameters

1. `curreny` - The base currency want to query, supported types is `CNY`, `USD`.

```js
params: ["CNY"]
```

##### Returns
- `currency` - The base currency, CNY or USD.
- `tokens` - Every token price int the currency.

##### Example
```js
// Request
curl -X GET --data '{"jsonrpc":"2.0","method":"loopring_getCutoff","params":{see above},"id":64}'

// Result
{
  "id":64,
  "jsonrpc": "2.0",
  "result": {
    "currency" : "CNY",
    "tokens" : [
        {
          "token": "ETH",
          "price": 31022.12 // hopeful price :)
        },
        {
          "token": "LRC",
          "price": 100.86
        }
     ]
  }
}
```
***

#### loopring_getEstimatedAllocatedAllowance

Get the total frozen amount of all unfinished orders

##### Parameters

1. `owner` - The address, if is null, will query all orders.
2. `token` - The specify token which you want to get.

```js
params: ["0x8888f1f195afa192cfee860698584c030f4c9db1", "WETH"]
```

##### Returns
- `string` - The frozen amount in hex format.

##### Example
```js
// Request
curl -X GET --data '{"jsonrpc":"2.0","method":"loopring_getEstimatedAllocatedAllowance","params":{see above},"id":64}'

// Result
{
  "id":64,
  "jsonrpc": "2.0",
  "result": {"0x2347ad6c"}
}
```
***

#### loopring_getSupportedMarket

Get relay supported all market pairs

##### Parameters
no input params.

```js
params: []
```

##### Returns
- `array of string` - The array of all supported markets.

##### Example
```js
// Request
curl -X GET --data '{"jsonrpc":"2.0","method":"loopring_getSupportedMarket","params":{see above},"id":64}'

// Result
{
  "id":64,
  "jsonrpc": "2.0",
  "result": ["SAN-WETH","GNO-WETH","RLC-WETH","AST-WETH"]
}
```
***

