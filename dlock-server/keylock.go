package main

import (
	"time"
)

type KeyLock struct {
	ClientId *string
	Created  time.Time
	Expires  time.Time // IsZero() means delete on disconnect

	waitCh chan bool
}

func NewKeyLock(clientId *string, now *time.Time, expires *time.Time) *KeyLock {
	kl := &KeyLock{
		ClientId: clientId,
		Created:  *now,
		waitCh:   make(chan bool, 1),
	}
	if expires != nil {
		kl.Expires = *expires
	}
	return kl
}

func (kl *KeyLock) CancelWait() {
	select {
	case <-kl.waitCh:
	default:
	}
}

func (k1 *KeyLock) IsSameClient(k2 *KeyLock) bool {
	return (k1.ClientId == k2.ClientId) ||
		(k1.ClientId != nil && *k1.ClientId == *k2.ClientId)
}

func (kl *KeyLock) Release() {
	select {
	case kl.waitCh <- true:
	default:
	}
}

func (kl *KeyLock) WaitTimeout(d time.Duration) (ok bool) {
	select {
	case ok = <-kl.waitCh:
		return ok
	case <-time.After(d):
		ok = false
	}
	return ok
}
