
# Relay API Spec V2.0

Loopring Relays are nodes that act as a bridge between Ethereum nodes and Loopring compatible wallets. A relay maintain global order-books for all trading pairs and is resposible for broadcasting orders selfishlessly to selected peer-to-peer networks. 

Wallets can host their own relay nodes to facility trading using Loopring, but can also take advantage of public relays provided by Loopring foundation or other third-parties. Order-book visulzation services or order browsers can also set up their own relay nodes to dispaly Loopring order-books to their users -- in such a senario, wallet-facing APIs can be disabled so the relay will run in a read-only mode. 

This document describes relay's public APIs v2.0 (JSON_RPC and SocketIO), but doesn't articulate how order-books nor trading history are maintained.

Against v1.0 supporting array and json request format, v2.0 unifies the request params to only support json format, and add socketIO support.

This document contains the following sections:
- Endport
- JSON-RPC Methods
- SocketIO Events


## Endport
```
JSON-RPC : http://{hostname}:{port}/rpc/v2/
JSON-RPC(mainnet) : https://relay1.loopring.io/rpc/v2/
Ethereum standard JSON-RPC : https://relay1.loopring.io/eth
SocketIO(local|test) : https://{hostname}:{port}/socket.io/
SocketIO(mainnet) : https://relay1.loopring.io/socket.io/
```

## JSON-RPC Methods 

* The relay supports all Ethereum standard JSON-RPCs, please refer to [eth JSON-RPC](https://github.com/ethereum/wiki/wiki/JSON-RPC).
* [loopring_getBalance](#loopring_getbalance)
* [loopring_submitOrder](#loopring_submitorder)
* [loopring_getOrders](#loopring_getorders)
* [loopring_getOrderByHash](#loopring_getorderbyhash)
* [loopring_getDepth](#loopring_getdepth)
* [loopring_getTicker](#loopring_getticker)
* [loopring_getTickers](#loopring_gettickers)
* [loopring_getFills](#loopring_getfills)
* [loopring_getTrend](#loopring_gettrend)
* [loopring_getRingMined](#loopring_getringmined)
* [loopring_getCutoff](#loopring_getcutoff)
* [loopring_getPriceQuote](#loopring_getpricequote)
* [loopring_getEstimatedAllocatedAllowance](#loopring_getestimatedallocatedallowance)
* [loopring_getGetFrozenLRCFee](#loopring_getgetfrozenlrcfee)
* [loopring_getSupportedMarket](#loopring_getsupportedmarket)
* [loopring_getSupportedTokens](#loopring_getsupportedtokens)
* [loopring_getContracts](#loopring_getcontracts)
* [loopring_getLooprSupportedMarket](#loopring_getlooprsupportedmarket)
* [loopring_getLooprSupportedTokens](#loopring_getlooprsupportedtokens)
* [loopring_getPortfolio](#loopring_getportfolio)
* [loopring_getTransactions](#loopring_gettransactions)
* [loopring_unlockWallet](#loopring_unlockwallet)
* [loopring_notifyTransactionSubmitted](#loopring_notifytransactionsubmitted)
* [loopring_submitRingForP2P](#loopring_submitringforp2p)

## SocketIO Events

* [portfolio](#portfolio)
* [balance](#balance)
* [tickers](#tickers)
* [loopringTickers](#loopringtickers)
* [transactions](#transactions)
* [marketcap](#marketcap)
* [depth](#depth)
* [trends](#trends)

## JSON RPC API Reference

#### loopring_getBalance

Get user's balance and token allowance info.

##### Parameters

- `owner` - The address, if is null, will query all orders.
- `delegateAddress` - The loopring [TokenTransferDelegate Protocol](https://github.com/Loopring/token-listing/blob/master/ethereum/deployment.md).

```js
params: [{
  "owner" : "0x847983c3a34afa192cfee860698584c030f4c9db1",
  "delegateAddress" : "0x5567ee920f7E62274284985D793344351A00142B"
}]
```

##### Returns

`Account` - Account balance info object.

- `delegateAddress` - The loopring [TokenTransferDelegate Protocol](https://github.com/Loopring/token-listing/blob/master/ethereum/deployment.md).
2. `tokens` - All token balance and allowance info array.

##### Example
```js
// Request
curl -X POST --data '{"jsonrpc":"2.0","method":"loopring_getBalance","params":{see above},"id":64}'

// Result
{
  "id":64,
  "jsonrpc": "2.0",
  "result": {
    "delegateAddress" : "0x5567ee920f7E62274284985D793344351A00142B",
    "tokens": [
      {
          "token": "LRC",
          "balance": "0x000001234d",
          "allowance": "0x0000001233a"
      },
      {
          "token": "WETH",
          "balance": "0x00000012dae734",
          "allowance": "0x00000012aae734"
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
  - `delegateAddress` - The loopring [TokenTransferDelegate Protocol](https://github.com/Loopring/token-listing/blob/master/ethereum/deployment.md).
  - `walletAddress` - The wallet margin address.
  - `owner` - user's wallet address
  - `AuthAddr` - The wallet auth public key.
  - `AuthPrivateKey` - The wallet auth private key to sign ring when submitting ring.
  - `tokenS` - Token to sell.
  - `tokenB` - Token to buy.
  - `amountS` - Maximum amount of tokenS to sell.
  - `amountB` - Minimum amount of tokenB to buy if all amountS sold.
  - `validSince` - Indicating when this order is created.
  - `validUntil` - How long, in seconds, will this order live.
  - `lrcFee` - Max amount of LRC to pay for miner. The real amount to pay is proportional to fill amount.
  - `buyNoMoreThanAmountB` - If true, this order does not accept buying more than `amountB`.
  - `marginSplitPercentage` - The percentage of savings paid to miner.
  - `v` - ECDSA signature parameter v.
  - `r` - ECDSA signature parameter r.
  - `s` - ECDSA signature parameter s.
  - `powNonce` - Order submitting must be verified by our pow check logic. If orders submitted exceeded in certain team, we will increase pow difficult.
  - `orderType` - The order type, enum is (market_order|p2p_order), default is market_order.

```js
params: [{
  "protocol" : "0x847983c3a34afa192cfee860698584c030f4c9db1",
  "delegateAddress" : "0x5567ee920f7E62274284985D793344351A00142B",
  "owner" : "0x847983c3a34afa192cfee860698584c030f4c9db1",
  "walletAddress" : "0x847983c3a34afa192cfee860698584c030f4c9db1",
  "authAddr" : "0xcE862ca5e8DE3c5258B05C558daFDC4B7703a217",
  "authPrivateKey" : "0xe84989447467e438565dd2715d93d7537e9bc07fe7dc3044d8cbf4bd10967a69",
  "tokenS" : "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
  "tokenB" : "0xEF68e7C694F40c8202821eDF525dE3782458639f",
  "amountS" : "0x0001234d234",
  "amountB" : "0x002a7d",
  "validSince" : "0x5af13e32",
  "valiUntil": "0x5af28fb2",
  "lrcFee" : "0x14",
  "buyNoMoreThanAmountB" : true,
  "marginSplitPercentage" : 50, // 0~100
  "v" : 112,
  "r" : "239dskjfsn23ck34323434md93jchek3",
  "s" : "dsfsdf234ccvcbdsfsdf23438cjdkldy",
  "powNonce" : 10,
  "orderType" : "market",
}]
```

##### Returns

`OrderHash` - The hash of the order.

##### Example
```js
// Request
curl -X POST --data '{"jsonrpc":"2.0","method":"loopring_submitOrder","params":{see above},"id":64}'

// Result
{
  "id":64,
  "jsonrpc": "2.0",
  "result": { "orderHash" : "0xc7756d5d556383b2f965094464bdff3ebe658f263f552858cc4eff4ed0aeafeb"}
}
```

***

#### loopring_getOrders

Get loopring order list.

##### Parameters

- `owner` - The address, if is null, will query all orders.
- `orderHash` - The order hash.
- `status` - order status enum string.(status collection is : ORDER_OPENED(include ORDER_NEW and ORDER_PARTIAL), ORDER_NEW, ORDER_PARTIAL, ORDER_FINISHED, ORDER_CANCEL, ORDER_CUTOFF)
- `delegateAddress` - The loopring [TokenTransferDelegate Protocol](https://github.com/Loopring/token-listing/blob/master/ethereum/deployment.md).
- `market` - The market of the order.(format is LRC-WETH)
- `side` - The side of order. only support "buy" and "sell".
- `orderType` - The type of order. only support "market_order" and "p2p_order", default is "market_order".
- `pageIndex` - The page want to query, default is 1.
- `pageSize` - The size per page, default is 50.

```js
params: [{
  "owner" : "0x847983c3a34afa192cfee860698584c030f4c9db1",
  "orderHash" : "0xf0b75ed18109403b88713cd7a1a8423352b9ed9260e39cb1ea0f423e2b6664f0",
  "status" : "ORDER_CANCEL",
  "side" : "buy",
  "orderType" : "market",
  "delegateAddress" : "0x5567ee920f7E62274284985D793344351A00142B",
  "market" : "coss-weth",
  "pageIndex" : 2,
  "pageSize" : 40
}]
```

##### Returns

`PageResult of Order` - Order list with page info

1. `data` 
  - `orginalOrder` - The original order info when submitting.(refer to [LoopringProtocol](https://github.com/Loopring/protocol/blob/master/contracts/LoopringProtocol.sol))
  - `status` - The current order status.
  - `dealtAmountS` - Dealt amount of token S.
  - `dealtAmountB` - Dealt amount of token B.
  - `cancelledAmountS` - cancelled amount of token S.
  - `cancelledAmountB` - cancelled amount of token B.

2. `total` - Total amount of orders.
3. `pageIndex` - Index of page.
4. `pageSize` - Amount per page.

##### Example
```js
// Request
curl -X POST --data '{"jsonrpc":"2.0","method":"loopring_getOrders","params":{see above},"id":64}'

// Result
{
  "id":64,
  "jsonrpc": "2.0",
  "result": {
    "data" : [
        {
             "originalOrder":{
                 "protocol":"0x8d8812b72d1e4ffCeC158D25f56748b7d67c1e78",
                 "delegateAddress":"0x17233e07c67d086464fD408148c3ABB56245FA64",
                 "address":"0x71C079107B5af8619D54537A93dbF16e5aab4900",
                 "hash":"0x52c90064a0503ce566a50876fc41e0d549bffd2ba757f859b1749a75be798819",
                 "tokenS":"LRC",
                 "tokenB":"WETH",
                 "amountS":"0x1b1ae4d6e2ef500000",
                 "amountB":"0xde0b6b3a7640000",
                 "validSince":"0x5aefd848",
                 "validUntil":"0x5af129c8",
                 "lrcFee":"0x19ac8532c2790000",
                 "buyNoMoreThanAmountB":false,
                 "marginSplitPercentage":"0x32",
                 "v":"0x1c",
                 "r":"0x8eb60e6b1ebfbb9ab7aaf1b54a78497f112cb1f6430cd414ffc2a1366639f35e",
                 "s":"0x1b65ca88a645d3540e8a89232b73e67818be5cd81c66fa0cc38802e7a8358226",
                 "walletAddress":"0xb94065482Ad64d4c2b9252358D746B39e820A582",
                 "authAddr":"0xEf04F928F89cFF2a86CB4C2086D2aDa7D3A29200",
                 "authPrivateKey":"0x94866e133eb0cc774ca09a9de59c4c671fee6f7e871104d5e14004ac46fcee2b",
                 "market":"LRC-WETH",
                 "side":"sell",
                 "createTime":1525667919
             },
             "dealtAmountS":"0x0",
             "dealtAmountB":"0x0",
             "cancelledAmountS":"0x0",
             "cancelledAmountB":"0x0",
             "status":"ORDER_OPENED",
        }
    ]
    "total" : 12,
    "pageIndex" : 1,
    "pageSize" : 10
  }
}
```

***

#### loopring_getOrderByHash

Get loopring order by order hash.

##### Parameters

- `orderHash` - The order hash.

```js
params: [{
  "orderHash" : "0xf0b75ed18109403b88713cd7a1a8423352b9ed9260e39cb1ea0f423e2b6664f0",
}]
```

##### Returns

`Object of Order` - Order detail info.

- `orginalOrder` - The original order info when submitting.(refer to [LoopringProtocol](https://github.com/Loopring/protocol/blob/master/contracts/LoopringProtocol.sol))
- `status` - The current order status.
- `dealtAmountS` - Dealt amount of token S.
- `dealtAmountB` - Dealt amount of token B.
- `cancelledAmountS` - cancelled amount of token S.
- `cancelledAmountB` - cancelled amount of token B.

##### Example
```js
// Request
curl -X POST --data '{"jsonrpc":"2.0","method":"loopring_getOrders","params":{see above},"id":64}'

// Result
{
  "id":64,
  "jsonrpc": "2.0",
  "result": {
     "originalOrder":{
         "protocol":"0x8d8812b72d1e4ffCeC158D25f56748b7d67c1e78",
         "delegateAddress":"0x17233e07c67d086464fD408148c3ABB56245FA64",
         "address":"0x71C079107B5af8619D54537A93dbF16e5aab4900",
         "hash":"0x52c90064a0503ce566a50876fc41e0d549bffd2ba757f859b1749a75be798819",
         "tokenS":"LRC",
         "tokenB":"WETH",
         "amountS":"0x1b1ae4d6e2ef500000",
         "amountB":"0xde0b6b3a7640000",
         "validSince":"0x5aefd848",
         "validUntil":"0x5af129c8",
         "lrcFee":"0x19ac8532c2790000",
         "buyNoMoreThanAmountB":false,
         "marginSplitPercentage":"0x32",
         "v":"0x1c",
         "r":"0x8eb60e6b1ebfbb9ab7aaf1b54a78497f112cb1f6430cd414ffc2a1366639f35e",
         "s":"0x1b65ca88a645d3540e8a89232b73e67818be5cd81c66fa0cc38802e7a8358226",
         "walletAddress":"0xb94065482Ad64d4c2b9252358D746B39e820A582",
         "authAddr":"0xEf04F928F89cFF2a86CB4C2086D2aDa7D3A29200",
         "authPrivateKey":"0x94866e133eb0cc774ca09a9de59c4c671fee6f7e871104d5e14004ac46fcee2b",
         "market":"LRC-WETH",
         "side":"sell",
         "createTime":1525667919
     },
     "dealtAmountS":"0x0",
     "dealtAmountB":"0x0",
     "cancelledAmountS":"0x0",
     "cancelledAmountB":"0x0",
     "status":"ORDER_OPENED",
  }
}
```

***

#### loopring_getDepth

Get depth and accuracy by token pair

##### Parameters

1. `market` - The market pair.
2 `delegateAddress` - The loopring [TokenTransferDelegate Protocol](https://github.com/Loopring/token-listing/blob/master/ethereum/deployment.md).
3. `length` - The length of the depth data. default is 20.


```js
params: [{
  "market" : "LRC-WETH",
  "delegateAddress": "0x5567ee920f7E62274284985D793344351A00142B",
  "length" : 10 // defalut is 50
}]
```

##### Returns

1. `depth` - The depth data, every depth element is a three length of array, which contain price, amount A and B in market A-B in order.
2. `market` - The market pair.
3. `delegateAddress` - The loopring [TokenTransferDelegate Protocol](https://github.com/Loopring/token-listing/blob/master/ethereum/deployment.md).

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
        ["0.0008666300","10000.0000000000","8.6663000000"]
      ],
      "sell" : [
        ["0.0008683300","900.0000000000","0.7814970000"],["0.0009000000","7750.0000000000","6.9750000000"],["0.0009053200","480.0000000000","0.4345536000"]
      ]
    },
    "market" : "LRC-WETH",
    "delegateAddress": "0x5567ee920f7E62274284985D793344351A00142B",
  }
}
```

***


#### loopring_getTicker

Get loopring 24hr merged tickers info from loopring relay.

##### Parameters
NULL


```js
params: [{}]
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
curl -X POST --data '{"jsonrpc":"2.0","method":"loopring_getTicker","params":[{see above}],"id":64}'

// Result
{
  "id":64,
  "jsonrpc": "2.0",
  "result": [{
    "exchange" : "",
    "market":"EOS-WETH",
    "high" : 30384.2,
    "low" : 19283.2,
    "last" : 28002.2,
    "vol" : 1038,
    "amount" : 1003839.32,
    "buy" : 122321,
    "sell" : 12388,
    "change" : "-50.12%"
  },
  {
    "exchange" : "",
    "market":"LRC-WETH",
    "high" : 30384.2,
    "low" : 19283.2,
    "last" : 28002.2,
    "vol" : 1038,
    "amount" : 1003839.32,
    "buy" : 122321,
    "sell" : 12388,
    "change" : "-50.12%"
  },
  {
    "exchange" : "",
    "market":"RDN-WETH",
    "high" : 30384.2,
    "low" : 19283.2,
    "last" : 28002.2,
    "vol" : 1038,
    "amount" : 1003839.32,
    "buy" : 122321,
    "sell" : 12388,
    "change" : "-50.12%"
  },
  {
    "exchange" : "",
    "market":"SAN-WETH",
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

#### loopring_getTickers

Get all market 24hr merged tickers info from loopring relay.

##### Parameters
1. `market` - The market info like LRC-WETH.


```js
params: [{
    "market" : "LRC-WETH"
}]
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
curl -X GET --data '{"jsonrpc":"2.0","method":"loopring_getTickers","params":{see above}},"id":64}'

// Result
{
  "id":64,
  "jsonrpc": "2.0",
  "result": {"loopr" : {
    "exchange" : "loopr",
    "high" : 30384.2,
    "low" : 19283.2,
    "last" : 28002.2,
    "vol" : 1038,
    "amount" : 1003839.32,
    "buy" : 122321,
    "sell" : 12388,
    "change" : "-50.12%"
  },
  "binance" : {
    "exchange" : "binance",
    "high" : 30384.2,
    "low" : 19283.2,
    "last" : 28002.2,
    "vol" : 1038,
    "amount" : 1003839.32,
    "buy" : 122321,
    "sell" : 12388,
    "change" : "-50.12%"
  },
  "okEx" : {
    "exchange" : "okEx",
    "high" : 30384.2,
    "low" : 19283.2,
    "last" : 28002.2,
    "vol" : 1038,
    "amount" : 1003839.32,
    "buy" : 122321,
    "sell" : 12388,
    "change" : "-50.12%"
  },
  "huobi" : {
    "exchange" : "huobi",
    "high" : 30384.2,
    "low" : 19283.2,
    "last" : 28002.2,
    "vol" : 1038,
    "amount" : 1003839.32,
    "buy" : 122321,
    "sell" : 12388,
    "change" : "-50.12%"
  }}
}
```

***

#### loopring_getFills

Get order fill history. This history consists of OrderFilled events.

##### Parameters

1. `market` - The market of the order.(format is LRC-WETH)
2. `owner` - The address, if is null, will query all orders.
3. `delegateAddress` - The loopring [TokenTransferDelegate Protocol](https://github.com/Loopring/token-listing/blob/master/ethereum/deployment.md).
4. `orderHash` - The order hash.
5. `ringHash` - The order fill related ring's hash.
6. `pageIndex` - The page want to query, default is 1.
7. `pageSize` - The size per page, default is 50.

```js
params: [{
  "market" : "LRC-WETH",
  "delegateAddress" : "0x5567ee920f7E62274284985D793344351A00142B",
  "owner" : "0x8888f1f195afa192cfee860698584c030f4c9db1",
  "orderHash" : "0xee0b482d9b704070c970df1e69297392a8bb73f4ed91213ae5c1725d4d1923fd",
  "ringHash" : "0x2794f8e4d2940a2695c7ecc68e10e4f479b809601fa1d07f5b4ce03feec289d5",
  "pageIndex" : 1,
  "pageSize" : 20 // max size is 50.
}]
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
  - `side` - Show the fill is Buy or Sell.
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

Get trend info per market.If you select interval 1Hr, this function will return a list(the length is 100 mostly). each item represent a data point of price change in 1Hr. The same for other intervals.

##### Parameters

1. `market` - The market type.
2. `interval` - The interval like 1Hr, 2Hr, 4Hr, 1Day, 1Week.

```js
params: {"market" : "LRC-WETH", "interval" : "2Hr"}

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
2. `delegateAddress` - The loopring [TokenTransferDelegate Protocol](https://github.com/Loopring/token-listing/blob/master/ethereum/deployment.md).
3. `pageIndex` - The page want to query, default is 1.
4. `pageSize` - The size per page, default is 50.

```js
params: [{
  "ringHash" : "0xb903239f8543d04b5dc1ba6579132b143087c68db1b2168786408fcbce568238",
  "delegateAddress" : "0x5567ee920f7E62274284985D793344351A00142B",
  "pageIndex" : 1,
  "pageSize" : 20 // max size is 50.
}]
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
curl -X GET --data '{"jsonrpc":"2.0","method":"loopring_getRingMined","params":{see above},"id":64}'

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
        "totalLrcFee" : "0x101",
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

1. `address` - The address.
2. `delegateAddress` - The loopring [TokenTransferDelegate Protocol](https://github.com/Loopring/token-listing/blob/master/ethereum/deployment.md).
3. `blockNumber` - "earliest", "latest" or "pending", default is "latest".

```js
params: [{
  "address": "0x8888f1f195afa192cfee860698584c030f4c9db1",
  "delegateAddress" : "0x5567ee920f7E62274284985D793344351A00142B",
  "blockNumber": "latest"
}]
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
params: [{ "currency" : "CNY" }]
```

##### Returns
- `currency` - The base currency, CNY or USD.
- `tokens` - Every token price int the currency.

##### Example
```js
// Request
curl -X GET --data '{"jsonrpc":"2.0","method":"loopring_getPriceQuote","params":{see above},"id":64}'

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

1. `owner` - The address.
2. `token` - The specify token which you want to get.
3. `delegateAddress` - The loopring [TokenTransferDelegate Protocol](https://github.com/Loopring/token-listing/blob/master/ethereum/deployment.md).

```js
params: [{
  "owner" : "0x8888f1f195afa192cfee860698584c030f4c9db1",
  "token" : "WETH",
  "delegateAddress" : "0x5567ee920f7E62274284985D793344351A00142B",
}]
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
  "result": "0x2347ad6c"
}
```
***

#### loopring_getGetFrozenLRCFee

Get the total frozen lrcFee of all unfinished orders

##### Parameters

1. `owner` - The address, if is null, will query all orders.
2. `delegateAddress` - The loopring [TokenTransferDelegate Protocol](https://github.com/Loopring/token-listing/blob/master/ethereum/deployment.md).

```js
params: [{
  "owner" : "0x8888f1f195afa192cfee860698584c030f4c9db1",
  "delegateAddress" : "0x5567ee920f7E62274284985D793344351A00142B",
}]
```

##### Returns
- `string` - The frozen amount in hex format.

##### Example
```js
// Request
curl -X GET --data '{"jsonrpc":"2.0","method":"loopring_getGetFrozenLRCFee","params":{see above},"id":64}'

// Result
{
  "id":64,
  "jsonrpc": "2.0",
  "result": "0x2347ad6c"
}
```
***

#### loopring_getSupportedMarket

Get relay supported all market pairs

##### Parameters
no input params.

```js
params: [{}]
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

#### loopring_getSupportedTokens

Get relay supported all tokens

##### Parameters
no input params.

```js
params: [{}]
```

##### Returns
- `array of string` - The array of all supported tokens.

##### Example
```js
// Request
curl -X GET --data '{"jsonrpc":"2.0","method":"loopring_getSupportedTokens","params":[{}],"id":64}'

// Result
{
  "id":64,
  "jsonrpc": "2.0",
  "result": [
      {
        "protocol":"0xd26114cd6EE289AccF82350c8d8487fedB8A0C07",
        "symbol":"OMG",
        "source":"omisego",
        "deny":false,
        "decimals":1000000000000000000,
        "isMarket":false
      },....
  ]
}
```
***

#### loopring_getContracts

Get relay supported all contracts. The result struct is map[delegateAddress] List(loopringProtocol)

##### Parameters
no input params.

```js
params: [{}]
```

##### Returns
- `json object` - The map of delegateAddress with list of loopringProtocol.

##### Example
```js
// Request
curl -X GET --data '{"jsonrpc":"2.0","method":"loopring_getContracts","params":[{}],"id":64}'

// Result
{
  "id":64,
  "jsonrpc": "2.0",
  "result": {
      "0x17233e07c67d086464fD408148c3ABB56245FA64": ["0x8d8812b72d1e4ffCeC158D25f56748b7d67c1e78"]
  }
}
```
***

#### loopring_getLooprSupportedMarket

Get Loopr wallet supported market pairs. Exactly same to loopring_getSupportedMarket but only method name.

#### loopring_getLooprSupportedTokens

Get Loopr wallet supported tokens. Exactly same to loopring_getSupportedTokens but only method name.

#### loopring_getPortfolio

Get user's portfolio info.

##### Parameters

- `owner` - The owner address.

```js
params: [{
  "owner" : "0x847983c3a34afa192cfee860698584c030f4c9db1"
}]
```

##### Returns

`Account` - Portfolio info object.

1. `tokens` - All token portfolio info array.

##### Example
```js
// Request
curl -X POST --data '{"jsonrpc":"2.0","method":"loopring_getPortfolio","params":{see above},"id":64}'

// Result
{
  "id":64,
  "jsonrpc": "2.0",
  "result": [
      {
          "token": "LRC",
          "amount": "0x000001234d",
          "percentage": "2.35"
      },
      {
          "token": "WETH",
          "amount": "0x00000012dae734",
          "percentage": "80.23"
      }
    ]
}
```

***

#### loopring_getTransactions

Get user's latest transactions by owner.

##### Parameters

- `owner` - The owner address, must be applied.
- `thxHash` - The transaction hash.
- `symbol` - The token symbol like LRC,WETH.
- `status` - The transaction status, enum is (pending|success|failed).
- `txType` - The transaction type, enum is (send|receive|enable|convert).
- `pageIndex` - The page want to query, default is 1.
- `pageSize` - The size per page, default is 10.


```js
params: [{
  "owner" : "0x847983c3a34afa192cfee860698584c030f4c9db1",
  "thxHash" : "0xc7756d5d556383b2f965094464bdff3ebe658f263f552858cc4eff4ed0aeafeb",
  "symbol" : "RDN",
  "status" : "pending",
  "txType" : "receive",
  "pageIndex" : 2, // default is 1
  "pageSize" : 20 // default is 20
}]
```

##### Returns

`PAGE RESULT of OBJECT`
1. `ARRAY OF DATA` - The transaction list.
  - `from` - The transaction sender.
  - `to` - The transaction receiver.
  - `owner` - the transaction main owner.
  - `createTime` - The timestamp of transaction create time.
  - `updateTime` - The timestamp of transaction update time.
  - `hash` - The transaction hash.
  - `blockNumber` - The number of the block which contains the transaction.
  - `value` - The amount of transaction involved.
  - `type` - The transaction type, like wrap/unwrap, transfer/receive.
  - `status` - The current transaction status.
2. `pageIndex`
3. `pageSize`
4. `total`

##### Example
```js
// Request
curl -X POST --data '{"jsonrpc":"2.0","method":"loopring_getTransactions","params":{see above},"id":64}'

// Result
{
  "id":64,
  "jsonrpc": "2.0",
  "result": {
      "data" : [
        {
          "owner":"0x66727f5DE8Fbd651Dc375BB926B16545DeD71EC9",
          "from":"0x66727f5DE8Fbd651Dc375BB926B16545DeD71EC9",
          "to":"0x23605cD09677600A91Df271C86E290cb09a17eeD",
          "createTime":150134131,
          "updateTime":150101931,
          "hash":"0xa226639a5852df7a61a19a473a5f6feb98be5247077a7b22b8c868178772d01e",
          "blockNumber":5029675,
          "value":"0x0000000a7640001",
          "type":"WRAP", // eth -> weth
          "status":"PENDING"
      }
    ],
    "pageIndex" : 1,
    "pageSize" : 20,
    "total" : 212
  }

}
```

***

#### loopring_unlockWallet

Tell the relay the unlocked wallet info.

##### Parameters

- `owner` - The address, if is null, will query all orders.

```js
params: [{
  "owner" : "0x847983c3a34afa192cfee860698584c030f4c9db1",
}]
```

##### Returns

`Account` - Account balance info object.

1. `string` - Success or fail info.

##### Example
```js
// Request
curl -X POST --data '{"jsonrpc":"2.0","method":"loopring_unlockWallet","params":{see above},"id":64}'

// Result
{
  "id":64,
  "jsonrpc": "2.0",
  "result": ["unlock_notice_success"]
}
```

***

#### loopring_notifyTransactionSubmitted

wallet should notify relay there was a transaction sending to eth network, then relay will get and save the pending transaction immediately.

##### Parameters

- `txHash` - The txHash.
- `nonce` - The owner newest nonce.
- `to` - The target address to send.
- `value` - The value in transaction.
- `gasPrice`.
- `gas`.
- `input` - The value input in transaction.
- `from` - The transaction sender.


```js
params: [{
    "hash":"0xb98c216fd29b627a2845a9c3eb6e2ac591049c07c71cd4e4c0f00962adfb4409",
    "nonce":"0x66",
    "to":"0x07a7191de1ba70dbe875f12e744b020416a5712b",
    "value":"0x16345785d8a0000",
    "gasPrice":"0x4e3b29200",
    "gas":"0x5208",
    "input":"0x",
    "from":"0x71c079107b5af8619d54537a93dbf16e5aab4900",
  }]
```

##### Returns

`String` - txHash.

1. no result if failed, you can see error info in param.

##### Example
```js
// Request
curl -X POST --data '{"jsonrpc":"2.0","method":"loopring_notifyTransactionSubmitted","params":{see above},"id":64}'

// Result
{
  "id":64,
  "jsonrpc": "2.0",
  "result": "0xb98c216fd29b627a2845a9c3eb6e2ac591049c07c71cd4e4c0f00962adfb4409"
}
```


***

#### loopring_submitRingForP2P

submit signed raw transaction of ring information, then relay can help submitting the ring while tracing the status of orders for wallet. 
please submit taker and maker order before invoke this method.

##### Parameters

- `takerOrderHash` - The taker order hash.
- `makerOrderHash` - The maker order hash.
- `rawTx` - The raw transaction.

```js
params: [{
  "takerOrderHash" : "0x52c90064a0503ce566a50876fc41e0d549bffd2ba757f859b1749a75be798819",
  "makerOrderHash" : "0x52c90064a0503ce566a50876fc41e0d549bffd2ba757f859b1749a75be798819",
  "rawTx" : "f889808609184e72a00082271094000000000000000000000000000000000000000080a47f74657374320000000000000000000000000000000000000000000000000000006000571ca08a8bbf888cfa37bbf0bb965423625641fc956967b81d12e23709cead01446075a01ce999b56a8a88504be365442ea61239198e23d1fce7d00fcfc5cd3b44b7215f",
}]
```

##### Returns

`txHash` - The transaction hash of eth_sendRawTransaction result. 

##### Example
```js
// Request
curl -X POST --data '{"jsonrpc":"2.0","method":"loopring_submitRingForP2P","params":{see above},"id":64}'

// Result
{
  "id":64,
  "jsonrpc": "2.0",
  "result": "0xf0458d1a96ed7678f3abfe469c754fcb974b79aa632fc7da246fa983f37a49ce"
}
```

***

## SocketIO Methods Reference

#### portfolio

Subscribe user's portfolio info by address.

##### subscribe events
- portfolio_req : emit this event to receive push message.
- portfolio_res : subscribe this event to receive push message.
- portfolio_end : emit this event to stop receive push message.

##### Parameters

- `owner` - The owner address.

```js
socketio.emit("portfolio_req", '{"owner" : "0x847983c3a34afa192cfee860698584c030f4c9db1"}', function(data) {
  // your business code
});
socketio.on("portfolio_res", function(data) {
  // your business code
});
```

##### Returns

`portfolios` - Portfolio info object.

1. `tokens` - All token portfolio info array.

##### Example
```js
// Request

'{"owner" : "0x847983c3a34afa192cfee860698584c030f4c9db1"}'

// Result
[
  {
    "token": "LRC",
    "amount": "0x000001234d",
    "percentage": 2.35
  },{
    "token": "WETH",
    "amount": "0x00000012dae734",
    "percentage": 80.23
  }
]
```
***

#### balance

Get user's balance and token allowance info.

##### subscribe events
- balance_req : emit this event to receive push message.
- balance_res : subscribe this event to receive push message.
- balance_end : emit this event to stop receive push message.

##### Parameters

- `owner` - The wallet address
- `delegateAddress` - The loopring [TokenTransferDelegate Protocol](https://github.com/Loopring/token-listing/blob/master/ethereum/deployment.md).

```js
socketio.emit("balance_req", '{"owner" : "0x847983c3a34afa192cfee860698584c030f4c9db1", "delegateAddress" : "0x5567ee920f7E62274284985D793344351A00142B"}', function(data) {
  // your business code
});
socketio.on("balance_res", function(data) {
  // your business code
});
```

##### Returns

`Account` - Account balance info object.

1. `delegateAddress` - The loopring [TokenTransferDelegate Protocol](https://github.com/Loopring/token-listing/blob/master/ethereum/deployment.md).
2. `tokens` - All token balance and allowance info array.

##### Example
```js
// Request
{
  "owner" : "0x847983c3a34afa192cfee860698584c030f4c9db1",
  "delegateAddress" : "0x5567ee920f7E62274284985D793344351A00142B"
}

// Result
{
    "delegateAddress" : "0x5567ee920f7E62274284985D793344351A00142B",
    "tokens": [
      {
          "token": "LRC",
          "balance": "0x000001234d",
          "allowance": "0x0000001233a"
      },
      {
          "token": "WETH",
          "balance": "0x00000012dae734",
          "allowance": "0x00000012aae734"
      }
    ]
}
```
***

#### loopringTickers

Get 24hr merged tickers info from loopring relay.

##### subscribe events
- loopringTickers_req : emit this event to receive push message.
- loopringTickers_res : subscribe this event to receive push message.
- loopringTickers_end : emit this event to stop receive push message.

##### Parameters
NULL

```js
socketio.emit("loopringTickers_req", '{}', function(data) {
  // your business code
});
socketio.on("loopringTickers_res", function(data) {
  // your business code
});
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

{}

// Result
[
  {
    "exchange" : "",
    "market" : "LRC-WETH",
    "high" : 30384.2,
    "low" : 19283.2,
    "last" : 28002.2,
    "vol" : 1038,
    "amount" : 1003839.32,
    "buy" : 122321,
    "sell" : 12388,
    "change" : "-50.12%"
  },
  {
    "exchange" : "",
    "market" : "RDN-WETH",
    "high" : 30384.2,
    "low" : 19283.2,
    "last" : 28002.2,
    "vol" : 1038,
    "amount" : 1003839.32,
    "buy" : 122321,
    "sell" : 12388,
    "change" : "-50.12%"
  },
  {
    "market" : "ZRX-WETH",
    "exchange" : "",
    "high" : 30384.2,
    "low" : 19283.2,
    "last" : 28002.2,
    "vol" : 1038,
    "amount" : 1003839.32,
    "buy" : 122321,
    "sell" : 12388,
    "change" : "-50.12%"
  },
  {
    "exchange" : "",
    "market" : "AUX-WETH"
    "high" : 30384.2,
    "low" : 19283.2,
    "last" : 28002.2,
    "vol" : 1038,
    "amount" : 1003839.32,
    "buy" : 122321,
    "sell" : 12388,
    "change" : "-50.12%"
  }
]
```

#### tickers

Get 24hr merged tickers reference info from other exchange like binance, huobi.

##### subscribe events
- tickers_req : emit this event to receive push message.
- tickers_res : subscribe this event to receive push message.
- tickers_end : emit this event to stop receive push message.

##### Parameters
1. `market` - The market selected.

```js
socketio.emit("tickers_req", '{"market" : "LRC-WETH"}', function(data) {
  // your business code
});
socketio.on("tickers_res", function(data) {
  // your business code
});
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

{"market" : "LRC-WETH"}

// Result
{
  "loopr" : {
    "exchange" : "loopr",
    "high" : 30384.2,
    "low" : 19283.2,
    "last" : 28002.2,
    "vol" : 1038,
    "amount" : 1003839.32,
    "buy" : 122321,
    "sell" : 12388,
    "change" : "-50.12%"
  },
  "binance" : {
    "exchange" : "binance",
    "high" : 30384.2,
    "low" : 19283.2,
    "last" : 28002.2,
    "vol" : 1038,
    "amount" : 1003839.32,
    "buy" : 122321,
    "sell" : 12388,
    "change" : "-50.12%"
  },
  "okEx" : {
    "exchange" : "okEx",
    "high" : 30384.2,
    "low" : 19283.2,
    "last" : 28002.2,
    "vol" : 1038,
    "amount" : 1003839.32,
    "buy" : 122321,
    "sell" : 12388,
    "change" : "-50.12%"
  },
  "huobi" : {
    "exchange" : "huobi",
    "high" : 30384.2,
    "low" : 19283.2,
    "last" : 28002.2,
    "vol" : 1038,
    "amount" : 1003839.32,
    "buy" : 122321,
    "sell" : 12388,
    "change" : "-50.12%"
  }
}
```

***

#### transaction

push user's latest 20 transactions by owner.

##### subscribe events
- transaction_req : emit this event to receive push message.
- transaction_res : subscribe this event to receive push message.
- transaction_end : emit this event to stop receive push message.

##### Parameters

- `owner` - The owner address.
- `thxHash` - The transaction hash.
- `symbol` - The token symbol, like LRC, WETH....
- `status` - The transaction status enum(pending, success, failed).
- `txType` - The transaction type(approve, send, receive, convert...).
- `pageIndex` - The pageIndex.
- `pageSize`  - The pageSize.

```js
socketio.emit("transaction_req", '{see below}', function(data) {
  // your business code
});
socketio.on("transaction_res", function(data) {
  // your business code
});
```

##### Returns

`PAGE RESULT of OBJECT`
1. `ARRAY OF DATA` - The transaction list.
  - `from` - The transaction sender.
  - `to` - The transaction receiver.
  - `owner` - the transaction main owner.
  - `createTime` - The timestamp of transaction create time.
  - `updateTime` - The timestamp of transaction update time.
  - `hash` - The transaction hash.
  - `blockNumber` - The number of the block which contains the transaction.
  - `value` - The amount of transaction involved.
  - `type` - The transaction type, like convert, transfer/receive.
  - `status` - The current transaction status.
2. `pageIndex`
3. `pageSize`
4. `total`

##### Example
```js
// Request
params: {
  "owner" : "0x847983c3a34afa192cfee860698584c030f4c9db1",
  "thxHash" : "0x2794f8e4d2940a2695c7ecc68e10e4f479b809601fa1d07f5b4ce03feec289d5",
  "symbol" : "WETH",
  "status" : "pending",
  "txType" : "receive",
  "pageIndex" : 1,
  "pageSize" : 20
}

// Result
[
  {
      "owner":"0x66727f5DE8Fbd651Dc375BB926B16545DeD71EC9",
      "from":"0x66727f5DE8Fbd651Dc375BB926B16545DeD71EC9",
      "to":"0x23605cD09677600A91Df271C86E290cb09a17eeD",
      "createTime":150134131,
      "updateTime":150101931,
      "hash":"0xa226639a5852df7a61a19a473a5f6feb98be5247077a7b22b8c868178772d01e",
      "blockNumber":5029675,
      "value":"0x0000000a7640001",
      "type":"convert", // eth -> weth
      "status":"PENDING"
  },{}...
]

```

***

#### marketcap

Get the USD/CNY/BTC quoted price of tokens.

##### subscribe events
- marketcap_req : emit this event to receive push message.
- marketcap_res : subscribe this event to receive push message.
- marketcap_end : emit this event to stop receive push message.

##### Parameters

1. `curreny` - The base currency want to query, supported types is `CNY`, `USD`.

```js
socketio.emit("marketcap_req", '{see below}', function(data) {
  // your business code
});
socketio.on("marketcap_res", function(data) {
  // your business code
});
```

##### Returns
- `currency` - The base currency, CNY or USD.
- `tokens` - Every token price int the currency.

##### Example
```js
// Request
{"currency" : "CNY"}

// Result
{
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
```
***

#### depth

Get depth and accuracy by token pair.

##### subscribe events
- depth_req : emit this event to receive push message.
- depth_res : subscribe this event to receive push message.
- depth_end : emit this event to stop receive push message.


##### Parameters

1. `market` - The market pair.
2. `delegateAddress` - The loopring [TokenTransferDelegate Protocol](https://github.com/Loopring/token-listing/blob/master/ethereum/deployment.md).
3. `length` - The length of the depth data. default is 20.


```js
socketio.emit("depth_req", '{see below}', function(data) {
  // your business code
});
socketio.on("depth_res", function(data) {
  // your business code
});
```

##### Returns

1. `depth` - The depth data, every depth element is a three length of array, which contain price, amount A and B in market A-B in order.
2. `market` - The market pair.
3. `delegateAddress` - The loopring [TokenTransferDelegate Protocol](https://github.com/Loopring/token-listing/blob/master/ethereum/deployment.md).

##### Example
```js
// Request
{
  "market" : "LRC-WETH",
  "delegateAddress" : "0x5567ee920f7E62274284985D793344351A00142B",
  "length" : 10 // defalut is 50
}

// Result
{
    "depth" : {
      "buy" : [
        ["0.0008666300","10000.0000000000","8.6663000000"]
      ],
      "sell" : [
        ["0.0008683300","900.0000000000","0.7814970000"],["0.0009000000","7750.0000000000","6.9750000000"],["0.0009053200","480.0000000000","0.4345536000"]
      ]
    },
    "market" : "LRC-WETH",
    "delegateAddress" : "0x5567ee920f7E62274284985D793344351A00142B",
  }
}
```

***

#### trends

Get trend info per market.

##### subscribe events
- trends_req : emit this event to receive push message.
- trends_res : subscribe this event to receive push message.
- trends_end : emit this event to stop receive push message.

##### Parameters

1. `market` - The market type.
2. `interval` - The interval like 1Hr, 2Hr, 4Hr, 1Day, 1Week default is 1Hr.
```js
params: {"market" : "LRC-WETH", "interval" : "1Hr"}

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
{"market" : "LRC-WETH", "interval" : "4hr"}


// Result
[
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
  }.{}....
]

```
