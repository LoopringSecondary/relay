# Loopring Relay
This program is under development.

## Set up
##### mysql
First,make sure mysql database have been installed which configured in relay/config/relay.toml

##### ipfs
relay need ipfs network to collect and broadcast orders,refer:<br>
https://ipfs.io/docs/install/

## install
install from source:
```
git clone https://github.com/Loopring/relay.git
```

## run
```
go run cmd/lrc/*
```