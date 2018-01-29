# Loopring Relay
The Loopring relay contains two parts: The Relay and Miner. The Relay is the service for wallet to broadcast orders to the ipfs network.  The Miner found ring from the unmatched order. It can act as one or both of them:<br>


**This program is still under development and heavy refactoring. We DO NOT RECOMMEND starting your own relay as major upgrades are expected through the first half of 2018.**

## Set up
##### etherenum node
The relay needs a full ethereum node in order to run. See ethererum documentation for details:<br>
https://github.com/ethereum/go-ethereum/wiki/Building-Ethereum

##### mysql
Mysql is the backing datastore. It needs to be installed, and the database needs to be configured as in relay/config/relay.toml

##### ipfs
Orders are collected and broadcast through the ipfs network. See ipfs documentation for details:<br>
https://ipfs.io/docs/install/

##### govendor
Install govendor to manage external golang packages
```
go get -u github.com/kardianos/govendor
```

## install the relay

build from source:
The environment variables: $GOROOT and $GOPATH must be set. 
```
> go get -u github.com/Loopring/relay
> cd $GOPATH/src/github.com/Loopring/relay
> make relay
```

## run as relay
```
> build/bin/relay --mode=relay
```


##run as miner
- step 1: You must have an eth account to sign and submit ring. Run `account ` to create or import it.
```
> build/bin/relay account --help
```
- step 2: You must modify the config file. Set `miner.miner` to the eth account which can be found in `keystore-dir`.
Then, you can run as follow.
```
> build/bin/relay  --mode=miner --unlocks $mineraddress --passwords $passwords

```
## docker
reference<br> 
https://hub.docker.com/r/loopring/relay
