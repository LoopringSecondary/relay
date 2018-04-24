package zklock

import (
	"github.com/go-zookeeper/zk"
	"github.com/Loopring/relay/config"
	"github.com/Loopring/relay/log"
	"fmt"
	"strings"
	"time"
)

type ZkLock struct {
	zkClient *zk.Conn
	lockMap map[string]*zk.Lock
}

const basePath = "/relay_lock"

func NewLock(options config.ZookeeperOptions) *ZkLock {
	if !options.WithZookeeper {
		return nil
	}
	if options.ZkServers == "" || len(options.ZkServers) < 10 {
		log.Fatalf("Zookeeper server list config invalid:%s", options.ZkServers)
	}
	zkClient, _, err := zk.Connect(strings.Split(options.ZkServers,","), time.Second * time.Duration(options.ConnectTimeOut))
	if err != nil {
		log.Fatalf("Connect zookeeper error:%s", err.Error())
	}
	return &ZkLock{zkClient, make(map[string]*zk.Lock)}
}

func (l *ZkLock) TryLock(lockName string) {
	if _, ok := l.lockMap[lockName]; !ok {
		acls := zk.WorldACL(zk.PermAll)
		l.lockMap[lockName] = zk.NewLock(l.zkClient, fmt.Sprintf("%s/%s", basePath, lockName), acls)
	}
	l.lockMap[lockName].Lock()
}

func (l *ZkLock) ReleaseLock(lockName string) {
	if innerLock, ok := l.lockMap[lockName]; ok {
		innerLock.Unlock()
	} else {
		log.Errorf("Try release not exists lock:%s", lockName)
	}
}