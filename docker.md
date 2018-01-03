## ****Relay****

Official relay implementation of the Ethereum protocol.


## **Building the source**
Building relay requires both a Go (version 1.8.3 or later) and a C compiler.
You can install them using your favourite package manager.
Once the dependencies are installed, clone code form git, and run
```bash
make relay
```

## **Global    options**
   --config value, -c value,            config file
   --mode,                              the mode that will be run, it can be set by relay, miner or full
   --unlocks,                           miner(feeRecipient) account to unlock
   --passwords,                         passwords used to unlock accounts
   --ringMaxLength, --rml,              the max length of ring
   --miner,                             the encrypted private key used to sign ring
   --feeRecepient, -r,                  the fee recepient address when mined a ring
   --ifRegistryRingHash, --reg          the submitter will registry ringhash first if it set ture
   --throwIfLrcIsInsuffcient, -t        the contract will revert when the lrc is insuffcient if it set ture
   --help, -h                           show help


## **Configuration**
As an alternative to passing the numerous flags to the relay binary, you can also pass a configuration file via:
--config /YOUR_CONFIG_PATH/docker.toml
quick start
```bash
relay --unlocks=YOUR_ACCOUNT --passwords=YOUR_PASSWORD --config=/YOUR_CONFIG_PATH/relay.toml --mode=miner
```

params in relay.toml
```$xslt
    log.zap_opts.output_paths              log path, absolute path, it should be /data/zap.log in docker container
    log.zap_opts.error_output_paths        error path, absolute path,it should be /data/err.log in docker container

    mysql.hostname                         mysql ip address, can use network alias in docker container,ex:mysql
    mysql.db_name                          create db before starting relay
    
    order_manager.cutoff_cache_expire_time cache of ordermanager cutoff address expire time
    order_manager.cutoff_cache_clean_time  cache of ordermanager cutoff address clean time, default 0(never clean)
    order_manager.dust_order_value         value of dust order which will be finished
    
    ipfs.server                            ipfs client ip address, can use network alias in docker container,ex:ipfs
    ipfs.listen_topics                     list of ipfs listen topics
    ipfs.broadcast_topics                  list of ipfs broadcast topics
    
    gateway.is_broadcast                   define whether relay will broadcast orders
    
    accessor.raw_url                       ethereum client http address,it can set by http:eth:8545 in docker container if network alias is eth
    
    common.default_block_number            value of started block on ethereum net.it should be the latest block on mainnet while started relay at the first time.
    common.save_event_log                  if this value is true, relay will save all transaction logs in mysql.
    common.protocolImpl.address            map of contracts version and address
    
    miner.feeRecipient                     feeRecipient
    miner.normal_miners.address            miner address

    keystore.keydir                        ethereum node keystore direction, in docker container you should mount it to the right direction: /keystore.
```

## **Creating docker image**
Install docker and run
```bash
docker build -t IMAGE_NAME:TAG RELAY_SOURCE_DIR
```

## **Docker quick start**
```bash
docker run -d --name relay -v YOUR_KEYSTORE_DIR:/keystore \
                           -v YOUR_DATA_DIR:/data \
                           -p 8083:8083 \
                           loopring/relay:v0.1-pre1 \
                           --unlocks=YOUR_MINER_ACCOUNT \
                           --passwords=YOUR_MIENR_PASSWORD \
                           --config=/data/docker.toml \
                           --mode=miner
```


## **Docker compose**
We have provide docker-compose.yaml, ref github.com/Loopring/relay/docker-compose.yml
Service relay depends on mysql,ipfs and ethereum client. mkdir with your favorite name ex prod and run 
```bash
docker-compose up
```
```bash
docker-compose ps
```
will pull images first, after all images downloaded, your will get lists of containers,such as prod_mysql_1,prod_eth_1,prod_ipfs_1,prod_relay_1.
try it again if containers failed or exited.

## **Attention**
Before start relay, we have to prepare token data and ethereum node.

**mysql**
Modify mysql root password(default value is example), and create database settled in relay.toml ex loorping_miner.
Although table lpr_tokens will be created while relay start, we suggest user create it earlier.

Sql for create table:
```sql
CREATE TABLE `lpr_tokens` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `protocol` varchar(42) DEFAULT NULL,
  `symbol` varchar(10) DEFAULT NULL,
  `source` varchar(200) DEFAULT NULL,
  `create_time` bigint(20) DEFAULT NULL,
  `deny` tinyint(1) DEFAULT NULL,
  `decimals` int(11) DEFAULT NULL,
  `is_market` tinyint(1) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uix_lpr_tokens_protocol` (`protocol`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8;
```

Sql for insert tokens:
```sql
INSERT INTO `lpr_tokens` VALUES ('1', '0x86fa049857e0209aa7d9e616f7eb3b3b78ecfdb0', 'EOS', 'eos', '0', '0', '18', '0'), ('2', '0xf230b790e05390fc8295f4d3f60332c93bed42e2', 'TRX', 'tron', '0', '0', '6', '0'), ('3', '0xd26114cd6EE289AccF82350c8d8487fedB8A0C07', 'OMG', 'omisego', '0', '0', '18', '0'), ('4', '0xd4fa1460f537bb9085d22c7bccb5dd450ef28e3a', 'PPT', 'populous', '0', '0', '8', '0'), ('5', '0x744d70fdbe2ba4cf95131626614a1763df805b9e', 'SNT', 'status', '0', '0', '18', '0'), ('6', '0xe94327d07fc17907b4db788e5adf2ed424addff6', 'REP', 'augur', '0', '0', '18', '0'), ('7', '0xB8c77482e45F1F44dE1745F52C74426C631bDD52', 'BNB', 'binance-coin', '0', '0', '18', '0'), ('8', '0x4156D3342D5c385a87D264F90653733592000581', 'SALT', 'salt', '0', '0', '8', '0'), ('9', '0xa74476443119A942dE498590Fe1f2454d7D4aC0d', 'GNT', 'golem-network-tokens', '0', '0', '18', '0'), ('10', '0x595832f8fc6bf59c85c527fec3740a1b7a361269', 'POWR', 'power-ledger', '0', '0', '6', '0'), ('11', '0xB97048628DB6B661D4C2aA833e95Dbe1A905B280', 'PAY', 'tenx', '0', '0', '18', '0'), ('12', '0xcB97e65F07DA24D46BcDD078EBebd7C6E6E3d750', 'BTM', 'bytom', '0', '0', '8', '0'), ('13', '0x0d8775f648430679a709e98d2b0cb6250d2887ef', 'BAT', 'batcoin', '0', '0', '18', '0'), ('14', '0xdd974d5c2e2928dea5f71b9825b8b646686bd200', 'KNC', 'kyber-network', '0', '0', '18', '0'), ('15', '0x5ca9a71b1d01849c0a95490cc00559717fcf0d1d', 'AE', 'aeternity', '0', '0', '18', '0'), ('16', '0xb7cb1c96db6b22b0d3d9536e0108d062bd488f74', 'WTC', 'walton', '0', '0', '18', '0'), ('17', '0x7C5A0CE9267ED19B22F8cae653F198e3E8daf098', 'SAN', 'santiment', '0', '0', '18', '0'), ('18', '0xe41d2489571d322189246dafa5ebde1f4699f498', 'ZRX', '0x', '0', '0', '18', '0'), ('19', '0x419D0d8BdD9aF5e606Ae2232ed285Aff190E711b', 'FUN', 'funfair', '0', '0', '8', '0'), ('20', '0x0F5D2fB29fb7d3CFeE444a200298f468908cC942', 'MANA', 'decentraland', '0', '0', '18', '0'), ('21', '0x255aa6df07540cb5d3d297f0d0d4d84cb52bc8e6', 'RDN', 'raiden-network-token', '0', '0', '18', '0'), ('22', '0x888666CA69E0f178DED6D75b5726Cee99A87D698', 'ICN', 'iconomi', '0', '0', '18', '0'), ('23', '0x6810e776880C02933D47DB1b9fc05908e5386b96', 'GNO', 'gnosis-gno', '0', '0', '18', '0'), ('24', '0x8f8221afbb33998d8584a2b05749ba73c37a938a', 'REQ', 'request-network', '0', '0', '18', '0'), ('25', '0x41e5560054824ea6b0732e656e3ad64e20e94e45', 'CVC', 'civic', '0', '0', '8', '0'), ('26', '0xB63B606Ac810a52cCa15e44bB630fd42D8d1d83d', 'MCO', 'monaco', '0', '0', '8', '0'), ('27', '0xF433089366899D83a9f26A773D59ec7eCF30355e', 'MTL', 'metal', '0', '0', '8', '0'), ('28', '0x1f573d6fb3f13d689ff844b4ce37794d79a7ff1c', 'BNT', 'bancor', '0', '0', '18', '0'), ('29', '0xc42209aCcC14029c1012fB5680D95fBd6036E2a0', 'PPP', 'paypie', '0', '0', '18', '0'), ('30', '0x5af2be193a6abca9c8817001f45744777db30756', 'ETHOS', 'ethos', '0', '0', '8', '0'), ('31', '0x08711D3B02C8758F2FB3ab4e80228418a7F8e39c', 'EDG', 'edgeless', '0', '0', '0', '0'), ('32', '0x514910771af9ca656af840dff83e8264ecf986ca', 'LINK', 'chainlink', '0', '0', '18', '0'), ('33', '0xB64ef51C888972c908CFacf59B47C1AfBC0Ab8aC', 'STORJ', 'storj', '0', '0', '8', '0'), ('34', '0xf970b8e36e23f7fc3fd752eea86f8be8d83375a6', 'RCN', 'ripio-credit-network', '0', '0', '18', '0'), ('35', '0x818fc6c2ec5986bc6e2cbf00939d90556ab12ce5', 'KIN', 'kin', '0', '0', '18', '0'), ('36', '0x99ea4db9ee77acd40b119bd1dc4e33e1c070b80d', 'QSP', 'quantstamp', '0', '0', '18', '0'), ('37', '0x960b236A07cf122663c4303350609A66A7B288C0', 'ANT', 'aragon', '0', '0', '18', '0'), ('38', '0xEF68e7C694F40c8202821eDF525dE3782458639f', 'LRC', 'loopring', '0', '0', '18', '0'), ('39', '0xaeC2E87E0A235266D9C5ADc9DEb4b2E29b54D009', 'SNGLS', 'singulardtv', '0', '0', '0', '0'), ('40', '0x9B11EFcAAA1890f6eE52C6bB7CF8153aC5d74139', 'ATM', 'attention-token-of-media', '0', '0', '8', '0'), ('41', '0x12fef5e57bf45873cd9b62e9dbd7bfb99e32d73e', 'CFI', 'cofound-it', '0', '0', '18', '0'), ('42', '0x667088b212ce3d06a1b553a7221E1fD19000d9aF', 'WINGS', 'wings', '0', '0', '18', '0'), ('43', '0x607F4C5BB672230e8672085532f7e901544a7375', 'RLC', 'rlc', '0', '0', '9', '0'), ('44', '0xf0ee6b27b759c9893ce4f094b49ad28fd15a23e4', 'ENG', 'enigma-project', '0', '0', '8', '0'), ('45', '0x40395044Ac3c0C57051906dA938B54BD6557F212', 'MGO', 'mobilego', '0', '0', '8', '0'), ('46', '0xAf30D2a7E90d7DC361c8C4585e9BB7D2F6f15bc7', '1ST', 'firstblood', '0', '0', '18', '0'), ('47', '0xcbcc0f036ed4788f63fc0fee32873d6a7487b908', 'HMQ', 'humaniq', '0', '0', '8', '0'), ('48', '0xBEB9eF514a379B997e0798FDcC901Ee474B6D9A1', 'MLN', 'melon', '0', '0', '18', '0'), ('49', '0x0abdace70d3790235af448c88547603b945604ea', 'DNT', 'district0x', '0', '0', '18', '0'), ('50', '0xf7b098298f7c69fc14610bf71d5e02c60792894c', 'GUP', 'guppy', '0', '0', '3', '0'), ('51', '0x27054b13b1B798B345b591a4d22e6562d47eA75a', 'AST', 'airswap', '0', '0', '4', '0'), ('52', '0xcb94be6f13a1182e4a4b6140cb7bf2025d28e41b', 'TRST', 'trust', '0', '0', '6', '0'), ('53', '0xE7775A6e9Bcf904eb39DA2b68c5efb4F9360e08C', 'TAAS', 'taas', '0', '0', '6', '0'), ('54', '0x1776e1f26f98b1a5df9cd347953a26dd3cb46671', 'NMR', 'numeraire', '0', '0', '18', '0'), ('55', '0xaaaf91d9b90df800df4f55c205fd6989c977e73a', 'TKN', 'tokencard', '0', '0', '8', '0'), ('56', '0xcfb98637bcae43C13323EAa1731cED2B716962fD', 'NET', 'nimiq', '0', '0', '18', '0'), ('57', '0xfa05A73FfE78ef8f1a739473e462c54bae6567D9', 'LUN', 'lunyr', '0', '0', '18', '0'), ('58', '0x4DF812F6064def1e5e029f1ca858777CC98D2D81', 'XAUR', 'xaurum', '0', '0', '8', '0'), ('59', '0xb9e7f8568e08d5659f5d29c4997173d84cdf2607', 'SWT', 'swarm-city', '0', '0', '18', '0'), ('60', '0xf05a9382A4C3F29E2784502754293D88b835109C', 'REX', 'real-estate-tokens', '0', '0', '18', '0'), ('61', '0x56ba2Ee7890461f463F7be02aAC3099f6d5811A8', 'CAT', 'blockcat', '0', '0', '18', '0'), ('62', '0x621d78f2ef2fd937bfca696cabaf9a779f59b3ed', 'DRP', 'dcorp', '0', '0', '2', '0'), ('63', '0xD8912C10681D8B21Fd3742244f44658dBA12264E', 'PLU', 'pluton', '0', '0', '18', '0'), ('64', '0xf8e386eda857484f5a12e4b5daa9984e06e73705', 'IND', 'indorse-token', '0', '0', '18', '0'), ('65', '0x2956356cD2a2bf3202F771F50D3D14A367b48070', 'WETH', 'ethereum', '0', '0', '18', '1');
```
if your are using private chain for test, just set lpr_tokens.protocol as your own.

**ethereum node**
We suggest user sync blocks from mainnet for relay ethereum client.
Mount your ethereum data to /root in container.
