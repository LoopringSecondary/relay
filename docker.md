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
   --throwIfLrcIsInsuffcient, -t        the contract will revert when the lrc is insuffcient if it set ture
   --help, -h                           show help


## **Configuration**
As an alternative to passing the numerous flags to the relay binary, you can also pass a configuration file via:
--config /YOUR_CONFIG_PATH/relay.toml
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
    
    market.token_file                      supported tokens and markets file
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
                           loopring/relay:v0.1.1 \
                           --unlocks=YOUR_MINER_ACCOUNT \
                           --passwords=YOUR_MIENR_PASSWORD \
                           --config=/data/relay.toml \
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
**ethereum node**
We suggest user sync blocks from mainnet for relay ethereum client.
While sync finished, before relay start, make sure default_block_number in relay.toml  is bigger than 4717454.

**tokens.json file**
tokens.json file listed tokens and markets those exchanges want to support.
you can combine markets like WETH-LRC

