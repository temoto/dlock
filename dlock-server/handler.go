package main

import (
	"github.com/temoto/dlock/dlock"
	"io"
	"time"
)

type HandlerFunc func(*Connection, *dlock.Request, io.Writer) error

func handleUnknown(conn *Connection, request *dlock.Request, w io.Writer) error {
	response := &dlock.Response{
		RequestId: request.Id,
		Status:    dlock.ResponseStatus_InvalidType.Enum(),
	}
	return dlock.SendMessage(w, response)
}

func handlePing(conn *Connection, request *dlock.Request, w io.Writer) error {
	response := &dlock.Response{
		RequestId: request.Id,
		Status:    dlock.ResponseStatus_Ok.Enum(),
	}
	return dlock.SendMessage(w, response)
}

func handleLock(conn *Connection, request *dlock.Request, w io.Writer) error {
	response := &dlock.Response{
		RequestId: request.Id,
		Status:    dlock.ResponseStatus_Ok.Enum(),
	}
	if request.Lock == nil || len(request.Lock.Keys) == 0 {
		response.Status = dlock.ResponseStatus_General.Enum()
		return dlock.SendMessage(w, response)
	}

	failKeys, err := conn.server.lockKeys(request.Lock.Keys, conn.keyLock(), time.Duration(request.Lock.GetWaitMicro())*time.Microsecond)
	response.Keys = failKeys
	if err == dlock.ErrorLockAcquireTimeout {
		response.Status = dlock.ResponseStatus_AcquireTimeout.Enum()
		return dlock.SendMessage(w, response)
	}
	if err != nil {
		response.Status = dlock.ResponseStatus_General.Enum()
		response.ErrorText = new(string)
		*response.ErrorText = err.Error()
	}

	return dlock.SendMessage(w, response)
}
