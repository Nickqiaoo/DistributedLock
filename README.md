# DistributedLock

## redis实现分布式锁

redis实现分布式锁的原理主要依靠redis执行命令的单线程操作，使用`SETNX`保证只有一个客户端可以将key设置成功，之后其他客户端监听key状态等待key被删除后尝试获得锁，在`SETNX`的官方文档下面给出了一个分布式锁的实现参考
```
SETNX lock.foo <current Unix time + lock timeout + 1>
```
使用`SETNX`获取锁，key是锁的名字，value是持有锁的时间戳，set成功获得锁，释放锁时del对应key，在持有锁时还应该对锁进行续期，如果没有set成功，则不断get，如果key不存在或者value已经过期，则`GETSET`设置新的时间戳，如果返回的时间戳未超时，说明有另一个客户端getset成功了，需要继续等待。之所以不能直接DEL之后SETNX是因为这里会有一个race condition：如果C1,C2两个客户端都get后发现key超时，C1执行DEL,SETNX获得锁，之后C2同样执行DEL,SETNX获得了锁，使用getset解决了这个问题。

这是文档给出的一种实现方法，这样实现有几个问题，一个是value设置的是时间戳，多机之间的时间可能有误差，还有如果续期更新超时时间失败，由于释放锁时直接DEL对应key，可能有其他客户端此时获得了锁，就会出现释放掉其他客户端获得的锁的问题，文档里还给出了一种单机的正确实现方法:

```
SET resource_name my_random_value NX PX 30000

if redis.call("get",KEYS[1]) == ARGV[1] then
    return redis.call("del",KEYS[1])
else
    return 0
end
```

value不再是时间戳，而是一个随机值，超时时间使用PX设置，释放锁时判断当前的value是否是之前设置的值，防止释放其他客户端获得的锁。

```go

```