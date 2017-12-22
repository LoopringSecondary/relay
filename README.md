# Loopring Relay
Loopring relay contains two partments:Relay and Miner. Relay is the service for wallet and broadcast orders to ipfs network ,Miner found ring from the unmatched orders. It can act as one or both of them:<br>

mention:
This program is under development.

## Set up
##### etherenum node
First,relay need a ethereum node,refer:<br>
https://github.com/ethereum/go-ethereum/wiki/Building-Ethereum

##### mysql
make sure mysql server have been installed,and database configured in relay/config/relay.toml

##### ipfs
relay need ipfs network to collect and broadcast orders,refer:<br>
https://ipfs.io/docs/install/

##### govendor
install govendor to manager external golang packages
```
go get -u github.com/kardianos/govendor
```

## install
install from source:
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


- step 1: You must have a eth account to sign and submit ring. Run `account ` to create or import it.
```
> build/bin/relay account --help
```
- step 2: You must modify config file. Set `miner.miner` to eth account which can be found in `keystore-dir`.
Then, you can run as follow.
```
> build/bin/relay  --mode=miner --unlocks $mineraddress --passwords $passwords

```
