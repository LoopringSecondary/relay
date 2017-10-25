package eth

import (
	"github.com/Loopring/ringminer/config"
	"github.com/Loopring/ringminer/db"
	"os"
	"strings"
	"testing"
)

func NewEthListener() *EthClientListener {
	l := &EthClientListener{}
	path := strings.TrimSuffix(os.Getenv("GOPATH"), "/") + "/src/github.com/Loopring/ringminer/config/ringminer.toml"
	globalConfig := config.LoadConfig(path)
	l.db = db.NewDB(globalConfig.Database)

	return l
}

func TestGetBlockNumber(t *testing.T) {
	l := NewEthListener()
	if number, err := l.getBlockNumber(); err != nil {
		t.Errorf(err.Error())
	} else {
		t.Log(number.String())
	}
}
