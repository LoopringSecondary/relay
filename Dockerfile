FROM golang:1.9-alpine as builder

RUN apk add --no-cache make gcc musl-dev linux-headers
RUN mkdir /data

ENV WORKSPACE=$GOPATH/src/github.com/Loopring/relay
ADD . $WORKSPACE

RUN cd $WORKSPACE && make relay
RUN mv $WORKSPACE/build/bin/relay /data

RUN ls -al /data

EXPOSE 8083 8545 3306 5001

ENTRYPOINT ["/data/relay"]