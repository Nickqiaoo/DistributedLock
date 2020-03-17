package distributedlock

// Lock is a distributedlock
type Lock interface{
	Lock() <-chan bool
	UnLock()
}