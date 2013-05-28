package main

import (
	"bufio"
	"github.com/temoto/dlock/dlock"
	"io"
	"log"
	"sort"
	"time"
)

type Connection struct {
	clientId     string
	handlers     map[dlock.RequestType]HandlerFunc
	messageCount uint64
	server       *Server
	socketFd     uintptr
	r            *bufio.Reader
	w            *bufio.Writer

	funClose             func() error
	funResetIdleTimeout  func() error
	funResetReadTimeout  func() error
	funResetWriteTimeout func() error
}

func (conn *Connection) keyLock() *KeyLock {
	return &KeyLock{
		ClientId: &conn.clientId,
		Created:  time.Now(),
		SocketFd: conn.socketFd,
	}
}

func (conn *Connection) loop() {
	defer conn.funClose()
	defer conn.server.releaseClient(conn.clientId)
	defer conn.server.wg.Done()

	for {
		conn.messageCount++
		request, err := conn.readRequest()
		if err != nil {
			// On EOF, close silently.
			if err == io.EOF {
				return
			}
			log.Printf("Connection.loop: %s #%d readRequest() error: %s\n",
				conn.clientId, conn.messageCount, err.Error())
			return
		}

		handler, ok := conn.handlers[request.GetType()]
		if !ok {
			handler = handleUnknown
		}

		conn.funResetWriteTimeout()
		err = handler(conn, request, conn.w)
		if err != nil {
			log.Printf("Connection.loop: %s #%d request.Id=%d request.Type=%s handler() error: %s",
				conn.clientId, conn.messageCount, request.GetId(), request.GetType().String(), err.Error())
			return
		}
		err = conn.w.Flush()
		if err != nil {
			log.Printf("Connection.loop: %s #%d request.Id=%d request.Type=%s Flush() error: %s",
				conn.clientId, conn.messageCount, request.GetId(), request.GetType().String(), err.Error())
			return
		}
	}
}

func (conn *Connection) readRequest() (*dlock.Request, error) {
	conn.funResetIdleTimeout()
	_, err := conn.r.Peek(4)
	if err != nil {
		return nil, err
	}

	request := &dlock.Request{}
	conn.funResetReadTimeout()
	err = dlock.ReadMessage(conn.r, request, conn.server.ConfigMaxMessage)

	if err == nil && request.Lock != nil && len(request.Lock.Keys) > 1 {
		sort.Strings(request.Lock.Keys)
	}

	return request, err
}
