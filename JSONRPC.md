
# Relay API Spec

Loopring Relays are nodes that act as a bridge between Ethereum nodes and Loopring compatible wallets. A relay maintain global order-books for all trading pairs and is resposible for broadcasting orders selfishlessly to selected peer-to-peer networks. 

Wallets can host their own relay nodes to facility trading using Loopring, but can also take advantage of public relays provided by Loopring foundation or other third-parties. Order-book visulzation services or order browsers can also set up their own relay nodes to dispaly Loopring order-books to their users -- in such a senario, wallet-facing APIs can be disabled so the relay will run in a read-only mode. 

This document describes relay's public APIs (JSON_RPC and WebSocket), but doesn't articulate how order-books nor trading history are maintained.

This document contains the following sections:
- Endport
- JSON-RPC Methods
- Websocket API


## Endport
```
JSON-RPC  : http://{hostname}:{port}/rpc
Websocket : wss://{hostname}:{port}/ws
```

## JSON-RPC Methods 

* The relay supports all Ethereum standard JSON-PRCs, please refer to [eth JSON-RPC](https://github.com/ethereum/wiki/wiki/JSON-RPC).
* [loopring_submitOrder](#loopring_submitorder)
* [loopring_getOrders](#loopring_getorders)
* [loopring_getDepth](#loopring_getdepth)
* [loopring_getTicker](#loopring_ticker)
* [loopring_getFills](#loopring_getfills)
* [loopring_getTrend](#loopring_getTrend)
* [loopring_getRingsMined](#loopring_getringsmined)
* [loopring_getCutoff](#loopring_getCutoff)
* [loopring_getCutoff](#loopring_getCutoff)

## Websocket APIs
TBD


## JSON RPC API Reference


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

`String` - The order hash.

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

`owner` - The address, if is null, will query all orders.
`status` - order status enum string.(status collection is : ORDER_NEW, ORDER_PARTIAL, ORDER_FINISHED, ORDER_CANCEL, ORDER_CUTOFF)
`contractVersion` - the loopring contract version you selected.
`market` - which market' order.(format is lrc-weth)
`pageIndex` - The page want to query, default is 1.
`pageSize` - The size per page, default is 50.

```js
params: {
  "owner" : "0x847983c3a34afa192cfee860698584c030f4c9db1",
  "status" : "ORDER_CANCEL",
  "contractVersion" : "v1.0",
  "market" : "coss-weth",
  "pageIndex" : 2,
  "pageSize" : 40
}
```

##### Returns

`PageResult of Order` - Order list with page info

1. `data` `LoopringOrder` - The original order info when submitting.(refer to [LoopringProtocol](https://github.com/Loopring/protocol/blob/master/contracts/LoopringProtocol.sol))
  - `address` - Order submit address
  - `protocol` - loopring protocol address
  - `tokenS` - Token to sell.
  - `tokenB` - Token to buy.
  - `amountS` - Maximum amount of tokenS to sell.
  - `amountB` - Minimum amount of tokenB to buy if all amountS sold.
  - `timestamp` - Indicating when this order is created.
  - `ttl` - How long, in seconds, will this order live.
  - `rand` - A random number to make this order's hash unique.
  - `lrcFee` - Max amount of LRC to pay for miner. The real amount to pay is proportional to fill amount.
  - `buyNoMoreThanAmountB` - If true, this order does not accept buying more than `amountB`.
  - `savingSharePercentage` - The percentage of savings paid to miner.
  - `v` - ECDSA signature parameter v.
  - `r` - ECDSA signature parameter r.
  - `s` - ECDSA signature parameter s.
  - `ts` - The submit TimeStamp.

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
      "rawOrder" : {
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
      "dealedAmountB" : "0x1a055690d9db80000",
      "dealedAmountS" : "0x1a055690d9db80000",
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
3. `length` - The length of the depth data. default is 50.


```js
params: {
  "market" : "Lrc-Weth",
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
    "market" : "Lrc-Weth",
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
params: {
    "contractVersion" : "v.10"
}
```

##### Returns

1. `high`
2. `low`
3. `last`
4. `vol`
5. `buy`
6. `sell`

##### Example
```js
// Request
curl -X GET --data '{"jsonrpc":"2.0","method":"loopring_ticker","params":[],"id":64}'

// Result
{
  "id":64,
  "jsonrpc": "2.0",
  "result": {
    "high" : 30384.2,
    "low" : 19283.2,
    "last" : 28002.2,
    "vol" : 1038,
    "buy" : 122321,
    "sell" : 12388,
    "ts" : 1506014710000
  }
}
```

***

#### loopring_getFills

Get order fill history. This history consists of OrderFilled events.

##### Parameters

1. `tokenS` - The token to sell
2. `tokenB` - The token to buy
3. `address`
4. `pageIndex`
5. `pageSize`

```js
params: {
  "tokenS" : "Eth",
  "tokenB" : "Lrc"
  "address" : "0x8888f1f195afa192cfee860698584c030f4c9db1",
  "pageIndex" : 1,
  "pageSize" : 20 // max size is 50.
}
```

##### Returns

`PAGE RESULT of OBJECT`
1. `ARRAY OF DATA` - The match histories.
  - `txHash` - The transaction hash of the match.
  - `fillAmountS` - Amount of sell.
  - `fillAmountB` - Amount of buy.
  - `ts` - The timestamp of matching time.
  - `relatedOrderHash` - The order hash.
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
        "txHash" : "0xb903239f8543d04b5dc1ba6579132b143087c68db1b2168786408fcbce568238",
        "fillAmountS" : 20,
        "fillAmountB" : 30.21,
        "ts" : 1506014710000
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

Get tick infos for kline.

##### Parameters

1. `market` - The token to sell

```js
params: {
  "from" : "Eth",
  "to" : "Lrc"
  "address" : "0x8888f1f195afa192cfee860698584c030f4c9db1",
  "pageIndex" : 1,
  "pageSize" : 20 // max size is 50.
```

##### Returns

`ARRAY of JSON OBJECT`
  - `fillAmountS` - Total amount of sell.
  - `fillAmountB` - Total amount of buy.
  - `ts` - The timestamp of matching time.
  - `open` - The opening price.
  - `close` - The closing price.
  - `high` - The highest price in interval.
  - `low` - The lowest price in interval.


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
        "fillAmountS" : 20,
        "fillAmountB" : 30.21,
        "ts" : 1506014710000
        "open" : 3232.1,
        "close" : 2321,
        "high" : 1231.2,
        "low" : 1234.2
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
2. `miner` - The miner that submit the ring.
3. `pageIndex` - The page want to query, default is 1.
4. `pageSize` - The size per page, default is 50.

```js
params: {
  "ringHash" : "0xb903239f8543d04b5dc1ba6579132b143087c68db1b2168786408fcbce568238",
  "miner" : "0x8888f1f195afa192cfee860698584c030f4c9db1"
  "pageIndex" : 1,
  "pageSize" : 20 // max size is 50.
}
```

##### Returns

1. `data` - The ring info.(refer to [Ring&RingMined](https://github.com/Loopring/protocol/blob/3bdc40c4f319e8fe70f58f82563db49579094b5c/contracts/LoopringProtocolImpl.sol#L109)
  - `ringHash` - The ring hash.
  - `miner` - The miner that submit match orders.
  - `feeRecepient` - The fee recepient address.
  - `orders` - The filled order list, the order struct refer to [OrderFilled](https://github.com/Loopring/protocol/blob/3bdc40c4f319e8fe70f58f82563db49579094b5c/contracts/LoopringProtocolImpl.sol#L117)
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
        "miner" : "0x8888f1f195afa192cfee860698584c030f4c9db1",
        "feeRecepient" : "0x8888f1f195afa192cfee860698584c030f4c9db1"
        "orders" : [
          {
            "orderHash" : "0xb903239f8543d04b5dc1ba6579132b143087c68db1b2168786408fcbce568238",
            "blockNumber" : 2345223,
            "prevOrderHash" : "0xb903239f8543d04b5dc1ba6579132b143087c68db1b2168786408fcbce568238",
            "nextOrderHash" : "0xb903239f8543d04b5dc1ba6579132b143087c68db1b2168786408fcbce568238",
            "amountS" : 34.2,
            "amountB" : 38.1,
            "lrcReward" : 0.2,
            "lrcFee" : 0.31
          }
        ]
       }
     ]
     "total" : 12,
     "pageIndex" : 1,
     "pageSize" : 10
  }
}
```
