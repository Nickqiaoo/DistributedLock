package etcdlock

import (
	"DistirbutedLock/conf"
	"context"
	"github.com/coreos/etcd/clientv3"
	"log"
)

//EtcdLock impl DistributedLock
type EtcdLock struct {
	etcd    *clientv3.Client
	leaseID clientv3.LeaseID
	name    string
	timeout int64
	get     chan bool
	cancel  context.CancelFunc
}

//Lock get lock
func (l *EtcdLock) Lock() <-chan bool {
	txn := clientv3.NewKV(l.etcd).Txn(context.TODO())
	lease := clientv3.NewLease(l.etcd)
	leaseResp, err := lease.Grant(context.TODO(), l.timeout)
	if err != nil {
		log.Println("lease grant:", err)
		l.get <- false
		return l.get
	}
	l.leaseID = leaseResp.ID
	ctx, cancel := context.WithCancel(context.TODO())
	l.cancel = cancel
	go l.leaseKeepAlive(ctx, l.leaseID, lease)

	txnResp, err := txn.If(clientv3.Compare(clientv3.CreateRevision(l.name), "=", 0)).
		Then(clientv3.OpPut(l.name, "test", clientv3.WithLease(l.leaseID))).
		Else().Commit()
	if err != nil {
		log.Println("txn:", err)
		l.get <- false
		return l.get
	}
	if !txnResp.Succeeded {
		log.Println("txnResp:", txnResp.Succeeded)
		go l.watchLock(ctx, txnResp.Header.GetRevision())
	} else {
		log.Println("lock success")
		l.get <- true
	}
	return l.get
}

//UnLock release lock
func (l *EtcdLock) UnLock() {
	log.Println("unlock")
	if l.cancel != nil {
		l.cancel()
	}
}

func (l *EtcdLock) leaseKeepAlive(ctx context.Context, leaseID clientv3.LeaseID, lease clientv3.Lease) {
	ch, err := lease.KeepAlive(ctx, leaseID)
	if err != nil {
		log.Println("lease keepalive:", err)
	} else {
		for res := range ch {
			log.Println("keepalive res:", res)
		}
	}
	//cancel后ch被关闭，释放lease
	if _, err := lease.Revoke(context.TODO(), leaseID); err != nil {
		log.Println("revoke err:", err)
	}
}

func (l *EtcdLock) watchLock(ctx context.Context, revision int64) {
	ch := l.etcd.Watch(ctx, l.name, clientv3.WithRev(revision + 1))

	for res := range ch {
		log.Println("watch res:", res)
		for _, ev := range res.Events {
			log.Println("events:", ev)
			if ev.Type == clientv3.EventTypeDelete {
				txn := clientv3.NewKV(l.etcd).Txn(context.TODO())
				txnResp, err := txn.If(clientv3.Compare(clientv3.CreateRevision(l.name), "=", 0)).
					Then(clientv3.OpPut(l.name, "test", clientv3.WithLease(l.leaseID))).
					Else().Commit()
				if err != nil {
					l.get <- false
					log.Println("txn:", err)
				}
				if txnResp.Succeeded {
					l.get <- true
					return
				} else {
					log.Println("txnResp:", txnResp.Succeeded)
				}
			}
		}
	}
}

//NewEtcdLock create etcdlock
func NewEtcdLock(c *conf.Etcd) *EtcdLock {
	client, err := clientv3.New(clientv3.Config{
		Endpoints:   c.Addr,
		DialTimeout: c.DialTimeout.Duration,
	})
	if err != nil {
		log.Println("create:", err)
	}
	return &EtcdLock{
		etcd:    client,
		name:    "lock",
		timeout: 100,
		get:     make(chan bool, 1),
	}
}
