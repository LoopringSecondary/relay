/*

  Copyright 2017 Loopring Project Ltd (Loopring Foundation).

  Licensed under the Apache License, Version 2.0 (the "License");
  you may not use this file except in compliance with the License.
  You may obtain a copy of the License at

  http://www.apache.org/licenses/LICENSE-2.0

  Unless required by applicable law or agreed to in writing, software
  distributed under the License is distributed on an "AS IS" BASIS,
  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
  See the License for the specific language governing permissions and
  limitations under the License.

*/

package ordermanager

import (
	"github.com/Loopring/relay/dao"
	"github.com/Loopring/relay/log"
	"github.com/Loopring/relay/marketcap"
	"github.com/Loopring/relay/types"
	"math/big"
)

type forkProcessor struct {
	dao dao.RdsService
	mc  marketcap.MarketCapProvider
}

func newForkProcess(rds dao.RdsService, mc marketcap.MarketCapProvider) *forkProcessor {
	processor := &forkProcessor{}
	processor.dao = rds
	processor.mc = mc

	return processor
}

func (p *forkProcessor) fork(event *types.ForkedEvent) error {
	log.Debugf("order manager processing chain fork......")

	from := event.ForkBlock.Int64()
	to := event.DetectedBlock.Int64()

	if err := p.dao.RollBackRingMined(from, to); err != nil {
		log.Errorf("order manager fork error:%s", err.Error())
	}
	if err := p.dao.RollBackFill(from, to); err != nil {
		log.Errorf("order manager fork error:%s", err.Error())
	}
	//if err := p.dao.RollBackCancel(from, to); err != nil {
	//	log.Errorf("order manager fork error:%s", err.Error())
	//}
	//if err := p.dao.RollBackCutoff(from, to); err != nil {
	//	log.Errorf("order manager fork error:%s", err.Error())
	//}

	// todo(fuk): isOrderCutoff???
	orderList, err := p.dao.GetOrdersWithBlockNumberRange(from, to)
	if err != nil {
		return err
	}

	forkBlockNumber := big.NewInt(from)
	for _, v := range orderList {
		state := &types.OrderState{}
		if err := v.ConvertUp(state); err != nil {
			log.Errorf("order manager fork error:%s", err.Error())
			continue
		}

		model, err := newOrderEntity(state, p.mc, forkBlockNumber)
		if err != nil {
			log.Errorf("order manager fork error:%s", err.Error())
			continue
		}

		model.ID = v.ID
		if err := p.dao.Save(model); err != nil {
			log.Debugf("order manager fork error:%s", err.Error())
			continue
		}
	}

	return nil
}
