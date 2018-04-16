#!/bin/bash
#AfterInstall

WORK_DIR=/opt/loopring/relay
SVC_DIR=/etc/service/relay
GOROOT=/usr/lib/go-1.9
export PATH=$PATH:$GOROOT/bin
export GOPATH=/opt/loopring/go-src

#cp svc config to svc
sudo cp -rf $WORK_DIR/src/bin/svc/* $SVC_DIR
sudo chmod -R 755 $SVC_DIR

SRC_DIR=$GOPATH/github.com/Loopring/relay
if [ ! -d $SRC_DIR ]; then
      sudo mkdir -p $SRC_DIR
fi

cd $SRC_DIR
sudo rm -rf ./*
sudo cp -r $WORK_DIR/src/* ./
sudo chmod -R 333 ./
go build -ldflags -s -v  -o build/bin/relay cmd/lrc/*
cp build/bin/relay $WORK_DIR/bin
