package redislock

import (
	"DistirbutedLock/conf"
	"github.com/gomodule/redigo/redis"
)

//RedisLock impl DistributedLock
type RedisLock struct {
	redis *redis.Pool
	name string
}

func (l *RedisLock) lock() {
	conn := l.redis.Get()
	defer conn.Close()
	if res, err := redis.Strings(conn.Do("MGET", args...)); err != nil {
		log.Errorf("conn.Do(MGET %v) error(%v)", args, err)
	}
}

func (l *RedisLock) unlock() {

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
