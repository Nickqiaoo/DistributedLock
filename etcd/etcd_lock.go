package etcdlock

import (
	"DistirbutedLock/conf"
	"log"
	"time"
	"github.com/coreos/etcd/clientv3"

)

//EtcdLock impl DistributedLock
type EtcdLock struct {
	etcd    *clientv3.Txn
	name    string
	timeout int64
	stop    chan bool
	get     chan bool
}
