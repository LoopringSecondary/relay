package market

import (
	"github.com/robfig/cron"
	"github.com/patrickmn/go-cache"
)

type Account struct {
	contractVersion string
	tokens []Balance
}

type Balance struct {
	token string
	balance string
	allowance string
}

type AccountManager struct {
	c             *cache.Cache
	cacheReady    bool
	cron		  *cron.Cron
}

func(a *AccountManager) getBalance(address string) Account {
	return Account{}
}

func(a *AccountManager) getNonceFromAccessor(address string) {

}

func(a *AccountManager) getCutoff(address string) {

}

func(a *AccountManager) getBalanceFromAccessor(address string) {

}



