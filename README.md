# Loopring Relay
Loopring relay can switch to relay or ringminer with config:<br>
the former may broadcast orders to ipfs network and provide series jsonrpc interfaces,<br>
ringminer will listen and extract ethereum network transactions,beside those,it's main task is matching orders to ring and send to ethereum network<br>

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