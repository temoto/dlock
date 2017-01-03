package main

import (
	"bufio"
	"github.com/temoto/dlock/dlock"
	"io"
	"log"
	"sort"
	"sync"
	"time"
)

type Connection struct {
	LastRequestTime time.Time
	Rch             chan *dlock.Request
	Wch             chan *dlock.Response

	clientId     string
	handlers     map[dlock.RequestType]HandlerFunc
	ioWait       sync.WaitGroup
	messageCount uint64
	r            *bufio.Reader
	server       *Server
	w            *bufio.Writer
	lk           sync.Mutex

	funClose             func() error
	funResetIdleTimeout  func() error
	funResetReadTimeout  func() error
	funResetWriteTimeout func() error
}

func NewConnection(server *Server, clientId string) *Connection {
	return &Connection{
		Rch: make(chan *dlock.Request, 1),
		Wch: make(chan *dlock.Response, 1),

		clientId: clientId,
		server:   server,
	}
}

func (conn *Connection) Wait() {
	conn.ioWait.Wait()
}

func (conn *Connection) keyLock() *KeyLock {
	now := time.Now()
	return NewKeyLock(&conn.clientId, &now, nil)
}

func (conn *Connection) loop() {
	defer conn.server.wg.Done()
	defer conn.server.releaseClient(&conn.clientId)
	defer conn.funClose()

	conn.ioWait.Add(2)
	go conn.readLoop()
	go conn.writeLoop()

	for request := range conn.Rch {
		handler, ok := conn.handlers[request.GetType()]
		if !ok {
			handler = handleUnknown
		}

		conn.funResetWriteTimeout()
		handler(conn, request)
		conn.server.profileTime("Connection.loop: handler", conn.LastRequestTime)
	}
	close(conn.Wch)

	conn.ioWait.Wait()
}

func (conn *Connection) readLoop() {
	defer conn.server.releaseClient(&conn.clientId)
	defer conn.ioWait.Done()
	defer close(conn.Rch)

	var err error
	for {
		conn.messageCount++

		conn.funResetIdleTimeout()
		_, err = conn.r.Peek(4)
		if err != nil {
			log.Printf("Connection.readLoop: %s #%d peek error: %s",
				conn.clientId, conn.messageCount, err.Error())
			return
		}

		request := &dlock.Request{}
		conn.funResetReadTimeout()
		err = dlock.ReadMessage(conn.r, request, conn.server.ConfigMaxMessage)

		if err == nil && request.Lock != nil && len(request.Lock.Keys) > 1 {
			sort.Strings(request.Lock.Keys)
		}
		if err != nil {
			// On EOF, close silently.
			if err == io.EOF {
				return
			}
			log.Printf("Connection.readLoop: %s #%d read error: %s",
				conn.clientId, conn.messageCount, err.Error())
			return
		}
		conn.lk.Lock()
		conn.LastRequestTime = time.Now()
		conn.lk.Unlock()
		conn.Rch <- request
	}
}

func (conn *Connection) writeLoop() {
	defer conn.ioWait.Done()

	var err error
	for response := range conn.Wch {
		conn.funResetWriteTimeout()
		err = dlock.SendMessage(conn.w, response)
		if err != nil {
			log.Printf("Connection.writeLoop: %s #%d response.RequestId=%d response.Status=%s SendMessage() error: %s",
				conn.clientId, conn.messageCount, response.GetRequestId(), response.GetStatus().String(), err.Error())
			return
		}

		err = conn.w.Flush()
		if err != nil {
			log.Printf("Connection.writeLoop: %s #%d response.RequestId=%d response.Status=%s Flush() error: %s",
				conn.clientId, conn.messageCount, response.GetRequestId(), response.GetStatus().String(), err.Error())
			return
		}
	}
}
