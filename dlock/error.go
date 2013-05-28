package dlock

import (
	"errors"
)

var (
	ErrorLockAcquireTimeout = errors.New("LockAcquireTimeout")
)
