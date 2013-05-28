package main

import (
	"bufio"
	"errors"
	"github.com/temoto/dlock/dlock"
	"log"
	"net"
	"strings"
	"sync"
	"time"
)

type Server struct {
	ConfigBind         string
	ConfigDebug        bool
	ConfigIdleTimeout  time.Duration
	ConfigMaxMessage   uint
	ConfigReadBuffer   uint
	ConfigReadTimeout  time.Duration
	ConfigWriteTimeout time.Duration

	clientLocks map[string][]string
	isClosed    bool
	keyLocks    map[string]*KeyLock
	listeners   []*net.TCPListener
	lk          sync.Mutex
	wg          sync.WaitGroup
}

var (
	ErrorIdleTimeout = errors.New("IdleTimeout")
	ErrorReadTimeout = errors.New("ReadTimeout")
)

func NewServer(bind string, timeout time.Duration) *Server {
	return &Server{
		ConfigBind:         bind,
		ConfigIdleTimeout:  timeout,
		ConfigReadTimeout:  timeout,
		ConfigWriteTimeout: timeout,
		ConfigMaxMessage:   16 << 10, // 16KB
		clientLocks:        make(map[string][]string),
		keyLocks:           make(map[string]*KeyLock),
	}
}

func (server *Server) Start() int {
	server.lk.Lock()
	defer server.lk.Unlock()

	for _, address := range strings.Split(server.ConfigBind, " ") {
		address := strings.TrimSpace(address)
		if address == "" {
			continue
		}

		tcpAddr, err := net.ResolveTCPAddr("tcp", address)
		if err != nil {
			log.Printf("Server.Start: ResolveTCPAddr: '%s' error: %s\n",
				address, err.Error())
			continue
		}
		listener, err := net.ListenTCP("tcp", tcpAddr)
		if err != nil {
			log.Printf("Server.Start: Error listening on '%s': %s\n", address, err.Error())
			continue
		}
		if server.ConfigDebug {
			log.Printf("Server.Start: bind to %s\n", listener.Addr().String())
		}

		server.listeners = append(server.listeners, listener)
		server.wg.Add(1)
		go server.listenLoop(listener)
	}
	return len(server.listeners)
}

func (server *Server) Close() {
	server.lk.Lock()
	defer server.lk.Unlock()
	if !server.isClosed {
		server.isClosed = true
		for _, listener := range server.listeners {
			listener.Close()
		}
	}
}

func (server *Server) Wait() {
	server.wg.Wait()
}

func (server *Server) get(key string) (*KeyLock, bool) {
	server.lk.Lock()
	kl, ok := server.keyLocks[key]
	server.lk.Unlock()
	return kl, ok
}

func (server *Server) listenLoop(l *net.TCPListener) {
	defer server.wg.Done()
	for {
		tcpConn, err := l.AcceptTCP()

		server.lk.Lock()
		if server.isClosed {
			break
		}
		server.lk.Unlock()

		if err != nil {
			log.Printf("Server.listenLoop: Accept() error: %s\n", err.Error())
			break
		}

		connection := &Connection{
			clientId: tcpConn.RemoteAddr().String(),
			handlers: map[dlock.RequestType]HandlerFunc{
				dlock.RequestType_Ping: handlePing,
				dlock.RequestType_Lock: handleLock,
			},
			server: server,

			funClose:             tcpConn.Close,
			funResetIdleTimeout:  func() error { return tcpConn.SetReadDeadline(time.Now().Add(server.ConfigIdleTimeout)) },
			funResetReadTimeout:  func() error { return tcpConn.SetReadDeadline(time.Now().Add(server.ConfigReadTimeout)) },
			funResetWriteTimeout: func() error { return tcpConn.SetWriteDeadline(time.Now().Add(server.ConfigWriteTimeout)) },
		}
		if server.ConfigDebug {
			log.Printf("Server.listenLoop: new connection: %s\n", connection.clientId)
		}

		if socket, err := tcpConn.File(); err != nil {
			log.Printf("Server.listenLoop: %s error: %s\n", connection.clientId, err.Error())
			return
		} else {
			connection.socketFd = socket.Fd()
		}
		if err := server.setupSocket(tcpConn); err != nil {
			log.Printf("Server.listenLoop: %s setupSocket error: %s\n", connection.clientId, err.Error())
			return
		}

		if server.ConfigReadBuffer == 0 {
			connection.r = bufio.NewReader(tcpConn)
		} else {
			connection.r = bufio.NewReaderSize(tcpConn, int(server.ConfigReadBuffer))
		}
		connection.w = bufio.NewWriter(tcpConn)

		server.wg.Add(1)
		go connection.loop()
	}
}

func (server *Server) lockKeys(keys []string, keyLock *KeyLock, timeout time.Duration) ([]string, error) {
	log.Printf("Server.lockKeys: < %s", *keyLock.ClientId)
	abort := false
	busyKeys := make([]string, 0, len(keys))
	result := make(chan error, 3)
	var someBusyKeyLock *KeyLock

	sleep := func() {
		time.Sleep(timeout)
		server.lk.Lock()
		defer server.lk.Unlock()
		if abort {
			return
		}
		abort = true
		result <- dlock.ErrorLockAcquireTimeout
	}

	try := func() bool {
		server.lk.Lock()
		defer server.lk.Unlock()
		if abort {
			return false
		}
		log.Printf("Server.lockKeys.try <")

		// Since we don't have transactional memory,
		// first: check if all requested keys are free
		busyKeys = busyKeys[:0]
		for _, key := range keys {
			if kl, ok := server.keyLocks[key]; ok && !kl.IsSameClient(keyLock) {
				busyKeys = append(busyKeys, key)
				someBusyKeyLock = kl
				continue
			}
		}
		log.Printf("Server.lockKeys.try: busykeys: %v", busyKeys)
		if len(busyKeys) > 0 {
			return true
		}

		// Then actually lock them.
		clientLocks, _ := server.clientLocks[*keyLock.ClientId]
		for _, key := range keys {
			keyLock.wg.Add(1)

			server.keyLocks[key] = keyLock

			// FIXME: check for dups
			clientLocks = append(clientLocks, key)
		}
		server.clientLocks[*keyLock.ClientId] = clientLocks

		abort = true
		result <- nil
		return false
	}

	wait := func() {
		const delayWait = 1000 * time.Millisecond
		const delayPoll = 10 * time.Millisecond
		for try() {
			if someBusyKeyLock != nil {
				someBusyKeyLock.WaitTimeout(delayWait)
			} else {
				time.Sleep(delayPoll)
			}
		}
	}

	if timeout != 0 {
		go sleep()
	}
	go wait()

	err := <-result
	return busyKeys, err
}

func (server *Server) releaseClient(clientId string) {
	if server.ConfigDebug {
		log.Printf("Server.releaseClient %s", clientId)
	}
	server.lk.Lock()
	keys, _ := server.clientLocks[clientId]
	delete(server.clientLocks, clientId)
	server.lk.Unlock()

	server.releaseKeys(keys)
}

func (server *Server) releaseKeys(keys []string) {
	if len(keys) == 0 {
		return
	}

	server.lk.Lock()
	for _, key := range keys {
		if kl, ok := server.keyLocks[key]; ok {
			kl.wg.Done()
		}
		delete(server.keyLocks, key)
	}
	server.lk.Unlock()
}

func (server *Server) setupSocket(conn *net.TCPConn) (err error) {
	if err = conn.SetLinger(0); err != nil {
		return
	}
	if server.ConfigReadBuffer != 0 {
		if err = conn.SetReadBuffer(int(server.ConfigReadBuffer)); err != nil {
			return
		}
	}
	if err = conn.SetKeepAlive(true); err != nil {
		return
	}
	if err = conn.SetReadDeadline(time.Now().Add(server.ConfigIdleTimeout)); err != nil {
		return
	}
	return
}
