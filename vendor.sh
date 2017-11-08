
# vendor init
rm -rf $GOPATH/src/github.com/Loopring/ringminer/vendor
govendor init

# vendor add external libraries
govendor add +external

# copy go-ethrenum c libs
rm -rf $GOPATH/src/github.com/Loopring/ringminer/vendor/github.com/ethereum/go-ethereum/crypto/secp256k1
cp -r $GOPATH/src/github.com/ethereum/go-ethereum/crypto/secp256k1 $GOPATH/src/github.com/Loopring/ringminer/vendor/github.com/ethereum/go-ethereum/crypto/
 
