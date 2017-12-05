package gateway

import (
	"fmt"
	"github.com/Loopring/relay/config"
	"github.com/Loopring/relay/types"
	"github.com/ipfs/go-ipfs-api"
)

type IPFSPubService interface {
	PublishOrder(order types.Order) error
}

type IPFSPubServiceImpl struct {
	options *config.IpfsOptions
	sh      *shell.Shell
}

func NewIPFSPubService(options *config.IpfsOptions) *IPFSPubServiceImpl {
	l := &IPFSPubServiceImpl{}

	l.options = options
	l.sh = shell.NewLocalShell()
	return l
}

func (p *IPFSPubServiceImpl) PublishOrder(order types.Order) error {
	orderJson, err := order.MarshalJSON()
	fmt.Println(orderJson)
	if err != nil {
		fmt.Println(err)
		return err
	}
	pubErr := p.sh.PubSubPublish(p.options.BroadcastTopics[0], string(orderJson))
	fmt.Println(pubErr)
	return pubErr
}
