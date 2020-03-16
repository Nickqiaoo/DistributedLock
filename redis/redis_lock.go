package redislock

import (
	"DistirbutedLock/conf"
	"log"
	"time"

	"github.com/gomodule/redigo/redis"
)

//RedisLock impl DistributedLock
type RedisLock struct {
	redis   *redis.Pool
	name    string
	timeout int64
	stop    chan bool
	get     chan bool
}

//Lock get lock
func (l *RedisLock) Lock() <-chan bool {
	conn := l.redis.Get()
	defer conn.Close()
	var res int64
	var err error
	if res, err = redis.Int64(conn.Do("SETNX", l.name, time.Now().Unix()+l.timeout)); err != nil {
		log.Println(res)
	}
	if res == 1 {
		go l.expire()
		l.get <- true
	} else if res == 0 {
		go l.check()
	}
	return l.get
}

//UnLock release lock
func (l *RedisLock) UnLock() {
	l.stop <- true
}

func (l *RedisLock) check() {
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case t := <-ticker.C:
			conn := l.redis.Get()
			defer conn.Close()
			var res int64
			var err error
			if res, err = redis.Int64(conn.Do("GET", l.name)); err != nil {
				log.Println(res)
			}
			if err == redis.ErrNil || res < time.Now().Unix(){
				if res, err = redis.Int64(conn.Do("GETSET", l.name, time.Now().Unix()+l.timeout)); err != nil {
					log.Println(res)
				}
				if err == redis.ErrNil || res < time.Now().Unix(){
					l.get <- true
					return
				}
			}
			log.Println("Current time: ", t)
		}
	}
}

func (l *RedisLock) expire() {
	ticker := time.NewTicker(time.Duration(l.timeout/2) * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-l.stop:
			log.Println("UnLock")
			conn := l.redis.Get()
			defer conn.Close()
			if res, err := conn.Do("DEL", l.name); err != nil {
				log.Println(res)
			}
			return
		case t := <-ticker.C:
			conn := l.redis.Get()
			defer conn.Close()
			if res, err := conn.Do("SET", l.name, time.Now().Unix() + l.timeout); err != nil {
				log.Println(res)
			}
			log.Println("Current time: ", t)
		}
	}
}

func newRedisLock(c *conf.Redis) *RedisLock {
	return &RedisLock{
		redis: &redis.Pool{
			MaxIdle:     c.Idle,
			MaxActive:   c.Active,
			IdleTimeout: c.IdleTimeout.Duration,
			Dial: func() (redis.Conn, error) {
				conn, err := redis.Dial(c.Network, c.Addr,
					redis.DialConnectTimeout(c.DialTimeout.Duration),
					redis.DialReadTimeout(c.ReadTimeout.Duration),
					redis.DialWriteTimeout(c.WriteTimeout.Duration),
					redis.DialPassword(c.Auth),
				)
				if err != nil {
					return nil, err
				}
				return conn, nil
			},
		},
	}
}