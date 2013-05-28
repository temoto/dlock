package main

import (
	"sync"
	"time"
)

type KeyLock struct {
	ClientId *string
	Created  time.Time
	SocketFd uintptr
	wg       sync.WaitGroup
}

func (k1 *KeyLock) IsSameClient(k2 *KeyLock) bool {
	return k1.SocketFd == k2.SocketFd && ((k1.ClientId == k2.ClientId) ||
		(k1.ClientId != nil && *k1.ClientId == *k2.ClientId))
}

func (kl *KeyLock) WaitTimeout(d time.Duration) bool {
	ch := make(chan bool, 2)
	go func() {
		kl.wg.Wait()
		ch <- true
	}()
	go func() {
		time.Sleep(d)
		ch <- false
	}()
	return <-ch
}
