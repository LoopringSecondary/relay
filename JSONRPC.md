
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
* [loopring_cancelOrder](#loopring_cancelorder)
* [loopring_getOrders](#loopring_getorders)
* [loopring_getDepth](#loopring_getdepth)
* [loopring_getTicker](#loopring_ticker)
* [loopring_getFills](#loopring_getfills)
* [loopring_getCandleTicks](#loopring_getcandleticks)
* [loopring_getRingsMined](#loopring_getringsmined)

## Websocket APIs
* [loopring_subscribeDepth](#loopring_subdepth)
* [loopring_subscribeCandleTick](#loopring_subscribecandletick)


## JSON RPC API Reference


***

#### loopring_submitOrder

Submit an order. The order is submitted to relay as a JSON object, this JSON will be broadcasted into peer-to-peer network for off-chain order-book maintainance and ring-ming. Once mined, the ring will be serialized into a transaction and submitted to Ethereum blockchain.

##### Parameters

`JSON Object` - The order object(refer to [LoopringProtocol](https://github.com/Loopring/protocol/blob/master/contracts/LoopringProtocol.sol))
  - `address` - Order submit address
  - `tokenS` - Token to sell.
  - `tokenB` - Token to buy.
  - `amountS` - Maximum amount of tokenS to sell.
  - `amountB` - Minimum amount of tokenB to buy if all amountS sold.
  - `timestamp` - Indicating when this order is created.
  - `ttl` - How long, in seconds, will this order live.
  - `salt` - A random number to make this order's hash unique.
  - `lrcFee` - Max amount of LRC to pay for miner. The real amount to pay is proportional to fill amount.
  - `buyNoMoreThanAmountB` - If true, this order does not accept buying more than `amountB`.
  - `savingSharePercentage` - The percentage of savings paid to miner.
  - `v` - ECDSA signature parameter v.
  - `r` - ECDSA signature parameter r.
  - `s` - ECDSA signature parameter s.

```js
params: {
  "address" : "0x847983c3a34afa192cfee860698584c030f4c9db1",
  "tokenS" : "Eth",
  "tokenB" : "Lrc",
  "amountS" : 100.3,
  "amountB" : 3838434,
  "timestamp" 1406014710,
  "ttl": 1200,
  "salt" : 3848348,
  "lrcFee" : 20,
  "buyNoMoreThanAmountB" : true,
  "savingSharePercentage" : 50, // 0~100
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
  "result": "0x47173285a8d7341e5e972fc677286384f802f8ef42a5ec5f03bbfa254cb01fad"
}
```

***

#### loopring_cancelOrder

Cancel an order.

##### Parameters

`JSON Object` - include order hash and signature params
  - `orderHash` - The order hash.
  - `v` - ECDSA signature parameter v.
  - `r` - ECDSA signature parameter r.
  - `s` - ECDSA signature parameter s.

```js
params: {
  "orderHash" : "0x47173285a8d7341e5e972fc677286384f802f8ef42a5ec5f03bbfa254cb01fad",
  "v" : 112,
  "r" : "239dskjfsn23ck34323434md93jchek3",
  "s" : "dsfsdf234ccvcbdsfsdf23438cjdkldy",
}
```

##### Returns

`String` - content like `SUBMIT_SUCCESS` for async request.

##### Example
```js
// Request
curl -X POST --data '{"jsonrpc":"2.0","method":"loopring_cancelOrder","params":{see above},"id":64}'

// Result
{
  "id":64,
  "jsonrpc": "2.0",
  "result": "SUBMIT_SUCCESS"
}
```

***

#### loopring_getOrderByHash

Get order details info by order hash.

##### Parameters

`String` - The order hash

```js
params: ["0x47173285a8d7341e5e972fc677286384f802f8ef42a5ec5f03bbfa254cb01fad"]
```

##### Returns

1. `JSON Object` - The original order info when submitting.(refer to [LoopringProtocol](https://github.com/Loopring/protocol/blob/master/contracts/LoopringProtocol.sol))
  - `address` - Order submit address
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

2. `status` `STRING` - Order status. refer to `Order Status Set` (include Pending, PartiallyExecuted, FullyExecuted, Cancelled)
3. `totalDealedAmountS` - The total amount of TokenS that have been selled. 
4. `totalDealedAmountB` - The total amount of TokenB that have been buyed.
5. `matchList` -  The match records related to this order.

##### Example
```js
// Request
curl -X POST --data '{"jsonrpc":"2.0","method":"loopring_getOrderByHash","params":{see above},"id":64}'

// Result
{
  "id":64,
  "jsonrpc": "2.0",
  "result": {
    "orginalOrder" : {
      "address" : "0x847983c3a34afa192cfee860698584c030f4c9db1",
      "tokenS" : "Eth",
      "tokenB" : "Lrc",
      "amountS" : 100.3,
      "amountB" : 3838434,
      "timestamp" : "2017-10-11 19:00:01",
      "ttl": 1200,
      "salt" : 3848348,
      "lrcFee" : 20,
      "buyNoMoreThanAmountB" : true,
      "savingSharePercentage" : 50, // 0~100
      "v" : 112,
      "r" : "239dskjfsn23ck34323434md93jchek3",
      "s" : "dsfsdf234ccvcbdsfsdf23438cjdkldy",
      "ts" : 1506014710000
    },
    "status" : "PartiallyExecuted",
    "totalDealedAmountS" : 30,
    "totalDealedAmountB" : 29333.21,
    "matchList" : {
      "total" : 301,
      "pageIndex" : 2,
      "pageSize" : 20
      "data" : [
        {
          "ts" : "1506014710000",
          "amountS" : 30.31,
          "amountB" : 3934.111,
          "txHash" : "0x1eb8d538bb9727028912f57c54776d90c1927e3b49f34a2e53e9271949ec044c"
        },
        {
          "ts" : "1506014710323",
          "amountS" : 30.31,
          "amountB" : 3934.111,
          "txHash" : "0x1eb8d538bb9727028912f57c54776d90c1927e3b49f34a2e53e9271949ec044c"
        }
      ]
    }
  }
}
```

***

#### loopring_getOrders

Get loopring order list.

##### Parameters

`address` - The address, if is null, will query all orders.
`status` - selected by status.
`pageIndex` - The page want to query, default is 1.
`pageSize` - The size per page, default is 50.

```js
params: {
  "address" : "0x847983c3a34afa192cfee860698584c030f4c9db1",
  "status" : "Canceled",
  "pageIndex" : 2,
  "pageSize" : 40
}
```

##### Returns

`PageResult of Order` - Order list with page info

1. `data` `LoopringOrder` - The original order info when submitting.(refer to [LoopringProtocol](https://github.com/Loopring/protocol/blob/master/contracts/LoopringProtocol.sol))
  - `address` - Order submit address
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
      "orginalOrder" : {
        "address" : "0x847983c3a34afa192cfee860698584c030f4c9db1",
        "tokenS" : "Eth",
        "tokenB" : "Lrc",
        "amountS" : 100.3,
        "amountB" : 3838434,
        "timestamp" : "2017-11-11 19:00:01",
        "ttl": 1200,
        "salt" : 3848348,
        "lrcFee" : 20,
        "buyNoMoreThanAmountB" : true,
        "savingSharePercentage" : 50, // 0~100
        "v" : 112,
        "r" : "239dskjfsn23ck34323434md93jchek3",
        "s" : "dsfsdf234ccvcbdsfsdf23438cjdkldy"
      },
      "status" : "PartiallyExecuted",
      "totalDealedAmountS" : 30,
      "totalDealedAmountB" : 29333.21,
      "matchList" : {
        "total" : 301,
        "pageIndex" : 2,
        "pageSize" : 20
        "data" : [
          {
            "ts" : "1506014710000",
            "amountS" : 30.31,
            "amountB" : 3934.111,
            "txHash" : "0x1eb8d538bb9727028912f57c54776d90c1927e3b49f34a2e53e9271949ec044c"
          },
          {
            "ts" : "1506014710323",
            "amountS" : 30.31,
            "amountB" : 3934.111,
            "txHash" : "0x1eb8d538bb9727028912f57c54776d90c1927e3b49f34a2e53e9271949ec044c"
          }
        ]
      },
      {}....
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

1. `tokenS` - The token to sell
2. `tokenB` - The token to buy
3. `length` - The length of the depth data. defalut is 50.


```js
params: {
  "tokenS" : "Eth",
  "tokenB" : "Lrc",
  "length" : 10 // defalut is 50
}
```

##### Returns

1. `depth` - The depth data.
2. `accuracies` - The accuracies, it's a array of number.

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
    "accuracies" : [0.01, 0.05, 0.1, 0.5]
  }
}
```

***


#### loopring_getTicker

Get 24hr merged ticker info from loopring relay.

##### Parameters

1. `tokenS` - The token to sell
2. `tokenB` - The token to buy

```js
params: {
  "from" : "Eth",
  "to" : "Lrc"
}
```

##### Returns

1. `high`
2. `low`
3. `last`
4. `vol`
5. `buy`
6. `sell`
7. `ts` - Timestamp.

##### Example
```js
// Request
curl -X GET --data '{"jsonrpc":"2.0","method":"loopring_ticker","params":{see above},"id":64}'

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

Get order fill history. This hisotry consists of OrderFilled events.

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

#### loopring_getCandleTicks

Get tick infos for kline.

##### Parameters

1. `tokenS` - The token to sell
2. `tokenB` - The token to buy
3. `interval` - The interval of kline. enum like: 1m, 5m, 6h, 1d....
4. `size` - The data size.

```js
params: {
  "from" : "Eth",
  "to" : "Lrc"
  "address" : "0x8888f1f195afa192cfee860698584c030f4c9db1",
  "pageIndex" : 1,
  "pageSize" : 20 // max size is 50.
}
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

***

## Websocket API Reference

***

#### loopring_subscribeDepth

subscribe depth data with websocket. after connected, client sends this message to server side.

##### Parameters

`JSON Object`
- `sub` - subscribe key. `market.depth.$tokenS.$tokenB`, tokenS and tokenB must be filled in lowercase.
- `id` - An identifier established by the client that MUST contain a number(same to json-rpc).

```js
{
  "sub": "market.depth.eth.lrc",
  "id": 64
}
```

##### Returns

`JSON Object`
- `sub` - subscribe key. `market.depth.$tokenS.$tokenB`, tokenS and tokenB must be filled in lowercase.
- `id` - An identifier established by the client that MUST contain a number(same to json-rpc).

##### Example
```js
// Send message
{
  "sub": "market.depth.eth.lrc",
  "id": 64
}

// Result
{
  "id": "64",
  "result": "SUB_SUCCESS",
  "message" : "" // if sub failed, this param contain error message.
}
```

***

***

#### loopring_subscribeCandleTick

subscribe candle tick data with websocket. after connected, client sends this message to server side. 

##### Parameters

`JSON Object`
- `sub` - subscribe key. `market.candle.$tokenS.$tokenB`, tokenS and tokenB must be filled in lowercase.
- `id` - An identifier established by the client that MUST contain a number(same to json-rpc).

```js
{
  "sub": "market.candle.eth.lrc",
  "id": 64
}
```

##### Returns

`JSON Object`
- `sub` - subscribe key. `market.candle.$tokenS.$tokenB`, tokenS and tokenB must be filled in lowercase.
- `id` - An identifier established by the client that MUST contain a number(same to json-rpc).

##### Example
```js
// Send message
{
  "sub": "market.candle.eth.lrc",
  "id": 64
}

// Result
{
  "id": "64",
  "result": "SUB_SUCCESS",
  "message" : "" // if sub failed, this param contain error message.
}
```

***
