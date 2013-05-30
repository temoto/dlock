package main

import (
	"bufio"
	"errors"
	"fmt"
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
	ErrorLockWaitAbort = errors.New("LockWaitAbort")

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
			log.Printf("Server.Start: ResolveTCPAddr: '%s' error: %s",
				address, err.Error())
			continue
		}
		listener, err := net.ListenTCP("tcp", tcpAddr)
		if err != nil {
			log.Printf("Server.Start: Error listening on '%s': %s", address, err.Error())
			continue
		}
		if server.ConfigDebug {
			log.Printf("Server.Start: bind to %s", listener.Addr().String())
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

func (server *Server) addConnection(tcpConn *net.TCPConn) *Connection {
	clientId := tcpConn.RemoteAddr().String()

	server.lk.Lock()
	if _, ok := server.clientLocks[clientId]; ok {
		log.Printf("Server.listenLoop: duplicate clientId: %s", clientId)
		return nil
	}
	server.clientLocks[clientId] = []string{}
	server.lk.Unlock()

	if err := server.setupSocket(tcpConn); err != nil {
		log.Printf("Server.listenLoop: %s setupSocket error: %s", clientId, err.Error())
		return nil
	}

	conn := NewConnection(server, clientId)
	conn.handlers = map[dlock.RequestType]HandlerFunc{
		dlock.RequestType_Ping: handlePing,
		dlock.RequestType_Lock: handleLock,
	}
	conn.funClose = tcpConn.Close
	conn.funResetIdleTimeout = func() error { return tcpConn.SetReadDeadline(time.Now().Add(server.ConfigIdleTimeout)) }
	conn.funResetReadTimeout = func() error { return tcpConn.SetReadDeadline(time.Now().Add(server.ConfigReadTimeout)) }
	conn.funResetWriteTimeout = func() error { return tcpConn.SetWriteDeadline(time.Now().Add(server.ConfigWriteTimeout)) }
	if server.ConfigDebug {
		log.Printf("Server.listenLoop: new conn: %s", conn.clientId)
	}

	if server.ConfigReadBuffer == 0 {
		conn.r = bufio.NewReader(tcpConn)
	} else {
		conn.r = bufio.NewReaderSize(tcpConn, int(server.ConfigReadBuffer))
	}
	conn.w = bufio.NewWriter(tcpConn)

	server.wg.Add(1)
	go conn.loop()

	return conn
}

func (server *Server) listenLoop(l *net.TCPListener) {
	defer server.wg.Done()
	for {
		tcpConn, err := l.AcceptTCP()
		if server.isClosed {
			break
		}
		if err != nil {
			log.Printf("Server.listenLoop: Accept() error: %s", err.Error())
			break
		}

		server.addConnection(tcpConn)
	}
}

func (server *Server) lockKeys(keys []string, keyLock *KeyLock, timeout time.Duration) ([]string, error) {
	defer server.profileTime(fmt.Sprintf("Server.lockKeys keys='%s' client=%s expires=%s timeout=%s",
		strings.Join(keys, " "), *keyLock.ClientId, keyLock.Expires, timeout), time.Now())
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
		if server.ConfigDebug {
			log.Printf("Server.lockKeys.try keys='%s' client=%s",
				strings.Join(keys, " "), *keyLock.ClientId)
		}
		server.lk.Lock()
		defer server.lk.Unlock()
		if abort {
			return false
		}

		// Client has disconnected; stop trying.
		if _, ok := server.clientLocks[*keyLock.ClientId]; !ok {
			log.Printf("Server.lockKeys.try keys='%s' client=%s disconnected",
				strings.Join(keys, " "), *keyLock.ClientId)
			abort = true
			return false
		}

		// Since we don't have transactional memory,
		// first, check if all requested keys are free
		busyKeys = busyKeys[:0]
		now := time.Now()
		for _, key := range keys {
			if kl, ok := server.unsafeTouchKey(key, &now); ok {
				if kl.IsSameClient(keyLock) {
					continue
				}
				busyKeys = append(busyKeys, key)
				someBusyKeyLock = kl
				continue
			}
		}
		if len(busyKeys) > 0 {
			return true
		}

		// Then actually lock them.
		clientLocks, _ := server.clientLocks[*keyLock.ClientId]
		for _, key := range keys {
			keyLock.CancelWait()

			server.keyLocks[key] = keyLock

			if stringListFind(clientLocks, key) == -1 {
				clientLocks = append(clientLocks, key)
			}
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
		result <- ErrorLockWaitAbort
	}

	if timeout != 0 {
		go sleep()
	}
	go wait()

	err := <-result
	return busyKeys, err
}

func (server *Server) profileTime(tag string, t1 time.Time) {
	d := time.Now().Sub(t1)
	if server.ConfigDebug {
		log.Printf("%s time=%s", tag, d)
	}
}

func (server *Server) releaseClient(clientId *string) []string {
	if server.ConfigDebug {
		log.Printf("Server.releaseClient: %s", *clientId)
	}
	server.lk.Lock()
	keys, _ := server.clientLocks[*clientId]
	delete(server.clientLocks, *clientId)
	server.lk.Unlock()

	server.releaseKeys(keys, nil)

	return keys
}

// With expire == nil would only remove keys that have Expires == nil
// otherwise would remove keys whoose Expires is less than the one passed.
func (server *Server) releaseKeys(keys []string, expire *time.Time) {
	if len(keys) == 0 {
		return
	}

	server.lk.Lock()
	defer server.lk.Unlock()
	for _, key := range keys {
		server.unsafeTouchKey(key, expire)
	}
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

// This function must be called while holding server.lk lock.
func (server *Server) unsafeDeleteKey(key string, kl *KeyLock) {
	if server.ConfigDebug {
		log.Printf("Server.unsafeDeleteKey key=%s kl.Expires=%s",
			key, kl.Expires)
	}
	delete(server.keyLocks, key)
	if kl != nil {
		if clientLocks, ok := server.clientLocks[*kl.ClientId]; ok {
			clientLocks = stringListRemove(clientLocks, key)
			if len(clientLocks) > 0 {
				server.clientLocks[*kl.ClientId] = clientLocks
			} else {
				delete(server.clientLocks, *kl.ClientId)
			}
		}
		kl.Release()
	}
}

// Releases the key if it is expired.
// This function must be called while holding server.lk lock.
func (server *Server) unsafeTouchKey(key string, expire *time.Time) (*KeyLock, bool) {
	if kl, ok := server.keyLocks[key]; ok {
		if server.ConfigDebug {
			log.Printf("Server.unsafeTouchKey key=%s expire=%s found; kl.Expires=%s",
				key, expire, kl.Expires)
		}
		if expire == nil && kl.Expires.IsZero() {
			server.unsafeDeleteKey(key, kl)
			return nil, false
		}
		if expire != nil && !kl.Expires.IsZero() && expire.Sub(kl.Expires) >= 0 {
			server.unsafeDeleteKey(key, kl)
			return nil, false
		}
		return kl, true
	}
	return nil, false
}
