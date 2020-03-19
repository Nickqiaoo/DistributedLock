package distributedlock

// Lock is a distributedlock
type Lock interface{
	Lock() <-chan bool //true for get the lock,false for error
	UnLock()
}