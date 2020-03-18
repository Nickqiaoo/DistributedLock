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
	name    string
	timeout int64
	stop    chan bool
	get     chan bool
	cancel  context.CancelFunc
}

//Lock get lock
func (l *EtcdLock) Lock() <-chan bool {
	txn := clientv3.NewKV(l.etcd).Txn(context.TODO())
	lease := clientv3.NewLease(l.etcd)
	leaseResp, err := lease.Grant(context.TODO(), l.timeout)
	if err != nil {
		log.Println(err)
		return l.get
	}
	leaseID := leaseResp.ID
	ctx, cancel := context.WithCancel(context.TODO())
	l.cancel = cancel
	go l.leaseKeepAlive(ctx, leaseID, lease)

	txn.If(clientv3.Compare(clientv3.CreateRevision(l.name), "=", 0)).
		Then(clientv3.OpPut(l.name, "", clientv3.WithLease(leaseID))).
		Else()
	txnResp, err := txn.Commit()
	if err != nil {
		log.Println(err)
		return l.get
	}
	if !txnResp.Succeeded {
		go l.watchLock()
	} else {
		l.get <- true
	}
	return l.get
}

//UnLock release lock
func (l *EtcdLock) UnLock() {
	l.cancel()
	l.stop <- true

}

func (l *EtcdLock) leaseKeepAlive(ctx context.Context, leaseID clientv3.LeaseID, lease clientv3.Lease) {
	ch, err := lease.KeepAlive(ctx, leaseID)
	if err != nil {
		log.Println(err)
	} else {
		for {
			select {
			case <-l.stop:
				if _, err := lease.Revoke(context.TODO(), leaseID); err != nil {
					log.Println(err)
				}
			case <-ch:
			}
		}
	}
}

func (l *EtcdLock) watchLock() {
	ch := l.etcd.Watch(context.TODO(), l.name)

	for {
		select {
		case res := <-ch:
			for _, ev := range res.Events {
				if ev.Type == clientv3.EventTypeDelete {
					txn.If(clientv3.Compare(clientv3.CreateRevision(l.name), "=", 0)).
						Then(clientv3.OpPut(l.name, "", clientv3.WithLease(leaseID))).
						Else()
					txnResp, err := txn.Commit()
					if err != nil {
						log.Println(err)
					}
					if txnResp.Succeeded {
						l.get <- true
						return
					}
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
		log.Println(err)
	}
	return &EtcdLock{
		etcd: client,
	}
}
