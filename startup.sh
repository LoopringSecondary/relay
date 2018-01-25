#!/bin/bash
echo "set config file..."
read debugfile

go run cmd/lrc/* --unlocks "0x4bad3053d574cd54513babe21db3f09bea1d387d" --passwords "101" --config $GOPATH/src/github.com/Loopring/relay/config/$debugfile --mode=miner

