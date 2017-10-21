package eth

import (
	ethch "github.com/Loopring/ringminer/chainclient/eth"
	"github.com/Loopring/ringminer/types"
	"math/big"
)

/*
该文件提供以下功能
1.保存blockhash key:blockhash,value:blockindex
2.查询blockindex
3.保存transaction key:blockhash,value:[]txhash
*/

//go:generate gencodec -type BlockIndex -field-override blockIndexMarshaling -out gen_blockindex_json.go
type BlockIndex struct {
	Number           *big.Int		`json:"number" 		gencodec:"required"`
	Hash             types.Hash		`json:"hash"		gencodec:"required"`
	ParentHash       string			`json:"parentHash"	gencodec:"required"`
}

type blockIndexMarshaling struct {
	Number 		*types.Big
}

type TransactionIndex struct {
	BlockTxs 	[]types.Hash 		`json:"blockTxs"	gencodec:"required"`
}

// 存储最近一次使用的blocknumber到db，同时存储blocknumber，blockhash键值对
func (l *EthClientListener) saveBlockInfo(block *ethch.Block) error {
	//key, err := blockNumberToBytes(block)
	//value := block.Hash
	//if err != nil {
	//	return err
	//}
	//l.db.Put()

	return nil
}

func (l *EthClientListener) getBlockInfo() {

}

func blockNumberToBytes(block *ethch.Block) ([]byte, error) {
	return block.Number.MarshalText()
}