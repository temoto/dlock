package main

import (
	"github.com/temoto/dlock/dlock"
	"time"
)

type HandlerFunc func(*Connection, *dlock.Request)

func commonResponse(conn *Connection, request *dlock.Request) *dlock.Response {
	return &dlock.Response{
		Version:        2,
		RequestId:      request.Id,
		Status:         dlock.ResponseStatus_Ok,
		ServerUnixTime: conn.LastRequestTime.UnixNano(),
	}
}

func handleUnknown(conn *Connection, request *dlock.Request) {
	response := commonResponse(conn, request)
	response.Status = dlock.ResponseStatus_InvalidType
	conn.Wch <- response
}

func handlePing(conn *Connection, request *dlock.Request) {
	response := commonResponse(conn, request)
	conn.Wch <- response
}

func handleLock(conn *Connection, request *dlock.Request) {
	response := commonResponse(conn, request)
	if request.Lock == nil || len(request.Lock.Keys) == 0 {
		response.Status = dlock.ResponseStatus_General
		conn.Wch <- response
		return
	}

	keyLock := conn.keyLock()
	if request.Lock.GetReleaseMicro() != 0 {
		keyLock.Expires = conn.LastRequestTime.Add(time.Duration(request.Lock.ReleaseMicro) * time.Microsecond)
	}
	waitTimeout := time.Duration(request.Lock.GetWaitMicro()) * time.Microsecond
	failKeys, err := conn.server.lockKeys(request.Lock.Keys, keyLock, waitTimeout)
	response.Keys = failKeys
	if err == dlock.ErrorLockAcquireTimeout {
		response.Status = dlock.ResponseStatus_AcquireTimeout
		conn.Wch <- response
		return
	}
	if err != nil {
		response.Status = dlock.ResponseStatus_General
		response.ErrorText = err.Error()
	}

	conn.Wch <- response
}
