package redislock

import (
	"DistirbutedLock/conf"
	"log"
	"math/rand"
	"time"

	"github.com/gomodule/redigo/redis"
)

const(
    LUA_SCRIPT = `
	if redis.call("get",KEYS[1]) == ARGV[1] then
    	return redis.call("del",KEYS[1])
	else
    	return 0
	end`
)

//RedisLock impl DistributedLock
type RedisLock struct {
	redis   *redis.Pool
	name    string
	value   int64
	timeout int64
	stop    chan bool
	get     chan bool
}

//Lock get lock
func (l *RedisLock) Lock() <-chan bool {
	conn := l.redis.Get()
	defer conn.Close()
	var res int
	var err error
	l.value = rand.Int63()
	if res, err = redis.Int(conn.Do("SET", l.name, l.value, "NX", "PX", l.timeout)); err != nil {
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
			var res int
			var err error
			if res, err = redis.Int(conn.Do("SET", l.name, l.value, "NX", "PX", l.timeout)); err != nil {
				log.Println(res)
			}
			if res == 1 {
				l.get <- true
				return
			}
			log.Println("Current time: ", t)
		case <-l.stop:
			return
		}
	}
}

func (l *RedisLock) expire() {
	ticker := time.NewTicker(time.Duration(l.timeout/2) * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-l.stop:
			log.Println("UnLock")
			conn := l.redis.Get()
			defer conn.Close()
			script := redis.NewScript(1, LUA_SCRIPT)
			if res, err := script.Do(conn, l.name, l.value); err != nil {
				log.Println(res)
			}
			l.redis.Close()
			return
		case t := <-ticker.C:
			conn := l.redis.Get()
			defer conn.Close()
			if res, err := conn.Do("PEXPIRE", l.name, l.timeout); err != nil {
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
