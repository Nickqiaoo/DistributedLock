package zklock

//ZkLock impl DistributedLock
type ZkLock struct {
	etcd    *clientv3.Client
	txn     clientv3.Txn
	leaseID clientv3.LeaseID
	name    string
	timeout int64
	stop    chan bool
	get     chan bool
	cancel  context.CancelFunc
}

