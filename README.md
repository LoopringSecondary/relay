
# Loopring Relay [DEPRECATED]
The Loopring relay contains two parts: The Relay and Miner. The Relay is the service for wallet to broadcast orders to the ipfs network.  The Miner found ring from the unmatched order. It can act as one or both of them:<br>


**THIS REPOSITORY HAS BEEN DEPRECATED. PLEASE USE OUR [relay-cluster](https://https://github.com/Loopring/relay-cluster) INSTEAD**

## SETUP
### Ethereum
The relay needs a full ethereum node in order to run. See ethererum documentation for details:<br>
https://github.com/ethereum/go-ethereum/wiki/Building-Ethereum

### MySQL
Mysql is the backing datastore. It needs to be installed, and the database needs to be configured as in relay/config/relay.toml

### IPFS
Orders are collected and broadcast through the ipfs network. See ipfs documentation for details:<br>
https://ipfs.io/docs/install/

### GOVENDOR
Install govendor to manage external golang packages
```
go get -u github.com/kardianos/govendor
```

## INSTALL

build from source:
The environment variables: $GOROOT and $GOPATH must be set. 
```
> go get -u github.com/Loopring/relay
> cd $GOPATH/src/github.com/Loopring/relay
> make relay
```

## RUN
```
> build/bin/relay --mode=relay
```


## RUN AS MINER
- step 1: You must have an eth account to sign and submit ring. Run `account ` to create or import it.
```
> build/bin/relay account --help
```
- step 2: You must modify the config file. Set `miner.miner` to the eth account which can be found in `keystore-dir`.
Then, you can run as follow.
```
> build/bin/relay  --mode=miner --unlocks $mineraddress --passwords $passwords

```
## DOCKER
reference<br> 
https://hub.docker.com/r/loopring/relay


