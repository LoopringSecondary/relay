package gateway

import (
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
	if err != nil {
		return err
	}
	return p.sh.PubSubPublish(p.options.BroadcastTopics[0], string(orderJson))
}
