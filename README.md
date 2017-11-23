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
```
git clone https://github.com/Loopring/relay.git
```

## run
```
go run cmd/lrc/*
```
