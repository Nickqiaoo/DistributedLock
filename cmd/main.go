package main

import (
	"DistirbutedLock/conf"
	"DistirbutedLock/etcd"
	"flag"
	"log"
	"time"
)

func main(){
	flag.Parse()
	if err := conf.Init(); err != nil {
		panic(err)
	}
	lock:=etcdlock.NewEtcdLock(conf.Conf.Etcd)
	get := lock.Lock()
	if <-get{
		log.Println("get lock")
	}else{
		log.Println("lock error")
	}
	lock.UnLock()
	time.Sleep(5)
}