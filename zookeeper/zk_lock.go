package zklock

import (
	"DistirbutedLock/conf"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/samuel/go-zookeeper/zk"
)

//ZkLock impl DistributedLock
type ZkLock struct {
	zk       *zk.Conn
	name     string
	lockpath string
	timeout  int64
	stop     chan bool
	get      chan bool
}

func parseSeq(path string) (int, error) {
	parts := strings.Split(path, "-")
	return strconv.Atoi(parts[len(parts)-1])
}

//Lock get lock
func (l *ZkLock) Lock() <-chan bool {
	lock := zk.NewLock(l.zk, "/lock", zk.AuthACL(zk.PermAll))
	lock.Lock()
	if l.lockpath != "" {
		l.get <- false
		return l.get
	}

	prefix := fmt.Sprintf("%s/lock-", l.name)

	path := ""
	var err error
	for i := 0; i < 3; i++ {
		path, err = l.zk.CreateProtectedEphemeralSequential(prefix, []byte{}, zk.AuthACL(zk.PermAll))
		if err == zk.ErrNoNode {
			// Create parent node.
			parts := strings.Split(l.name, "/")
			pth := ""
			for _, p := range parts[1:] {
				var exists bool
				pth += "/" + p
				exists, _, err = l.zk.Exists(pth)
				if err != nil {
					l.get <- false
					return l.get
				}
				if exists == true {
					continue
				}
				_, err = l.zk.Create(pth, []byte{}, 0, zk.AuthACL(zk.PermAll))
				if err != nil && err != zk.ErrNodeExists {
					l.get <- false
					return l.get
				}
			}
		} else if err == nil {
			break
		} else {
			l.get <- false
			return l.get
		}
	}
	if err != nil {
		l.get <- false
		return l.get
	}

	seq, err := parseSeq(path)
	if err != nil {
		l.get <- false
		return l.get
	}
	go l.watchLock(seq, path)
	return l.get
}

func (l *ZkLock) watchLock(seq int, path string) {
	for {
		children, _, err := l.zk.Children(l.name)
		if err != nil {
			l.get <- false
			return
		}

		lowestSeq := seq
		prevSeq := -1
		prevSeqPath := ""
		for _, p := range children {
			s, err := parseSeq(p)
			if err != nil {
				l.get <- false
			}
			if s < lowestSeq {
				lowestSeq = s
			}
			if s < seq && s > prevSeq {
				prevSeq = s
				prevSeqPath = p
			}
		}

		if seq == lowestSeq {
			// Acquired the lock
			l.get <- true
			l.lockpath = path
			break
		}

		// Wait on the node next in line for the lock
		_, _, ch, err := l.zk.GetW(l.name + "/" + prevSeqPath)
		if err != nil && err != zk.ErrNoNode {
			l.get <- false
			return
		} else if err != nil && err == zk.ErrNoNode {
			// try again
			continue
		}
		select {
		case ev := <-ch:
			if ev.Err != nil {
				l.get <- false
				return
			}
		case <-l.stop:
			return
		}
	}
}

//UnLock release lock
func (l *ZkLock) UnLock() {
	l.stop <- true
	if l.lockpath == "" {
		return
	}
	if err := l.zk.Delete(l.lockpath, -1); err != nil {
		log.Println("delete lock error:", err)
	}
}

//NewZkLock create zklock
func NewZkLock(c *conf.Zk) *ZkLock {
	client, _, err := zk.Connect(c.Addr, c.DialTimeout.Duration)
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
