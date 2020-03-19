package zklock

import (
	"DistirbutedLock/conf"
	"github.com/samuel/go-zookeeper/zk"
)

//ZkLock impl DistributedLock
type ZkLock struct {
	zk      *zk.conn
	name    string
	timeout int64
	stop    chan bool
	get     chan bool
	cancel  context.CancelFunc
}

//Lock get lock
func (l *ZkLock) Lock() <-chan bool {
	lock:=zk.NewLock(l.zk,"/lock")
	lock.Lock()
}

//NewZkLock create zklock
func NewZkLock(c *conf.Zk) *ZkLock {
	client, _, err := zk.Connect(c.Addr, c.DialTimeout)
	if err != nil {
		panic(err)
	}
	return &ZkLock{
		zk:      client,
		name:    "lock",
		timeout: 100,
		get:     make(chan bool, 1),
		stop:    make(chan bool, 1),
	}
}
