FROM golang:1.9-alpine as builder

RUN apk add --no-cache make gcc musl-dev linux-headers
RUN mkdir /data
RUN mkdir /keystore

ENV WORKSPACE=$GOPATH/src/github.com/Loopring/relay
ADD . $WORKSPACE

RUN cd $WORKSPACE && make relay
RUN mv $WORKSPACE/build/bin/relay /$GOPATH/bin

EXPOSE 8083

ENTRYPOINT ["relay"]