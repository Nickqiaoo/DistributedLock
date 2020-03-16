package distributedlock

// Lock is a distributedlock
type Lock interface{
	lock()
	unlock()
}