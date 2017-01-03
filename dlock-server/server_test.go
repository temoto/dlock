package main

import (
	"github.com/temoto/dlock/dlock"
	"log"
	"net"
	"testing"
	"time"
)

func assertNil(err error) {
	if err != nil {
		panic(err)
	}
}

func initTestServer(t *testing.T, timeout time.Duration) *Server {
	server := NewServer(":0", timeout)
	server.ConfigDebug = true

	n := server.Start()
	if n != 1 {
		t.Fatal("initTestServer: expected 1 listener, Server.Start():", n)
	}

	return server
}

func TestDuplicateClient(t *testing.T) {
	server := initTestServer(t, 10*time.Millisecond)

	netConn, err := net.Dial("tcp", server.listeners[0].Addr().String())
	assertNil(err)
	tcpConn := netConn.(*net.TCPConn)

	// Let the server accept new connection
	time.Sleep(1 * time.Millisecond)

	if server.addConnection(tcpConn) == nil {
		t.Fatal("Must add connection first time")
	}
	if server.addConnection(tcpConn) != nil {
		t.Fatal("Must not add same connection twice")
	}

	ch := make(chan bool)
	go func() {
		server.Close()
		server.Wait()
		ch <- true
	}()
	select {
	case <-ch:
	case <-time.After(11 * time.Millisecond):
		t.Fatal("Close timeout. Deadlock?")
	}
}

func TestFunctionalLock01(t *testing.T) {
	server := initTestServer(t, 10*time.Millisecond)

	conn1, err := net.Dial("tcp", server.listeners[0].Addr().String())
	assertNil(err)
	err = conn1.SetDeadline(time.Now().Add(100 * time.Millisecond))
	assertNil(err)

	request1 := &dlock.Request{
		Type: dlock.RequestType_Lock,
		Lock: &dlock.RequestLock{
			Keys: []string{"q"},
		},
	}
	err = dlock.SendMessage(conn1, request1)
	assertNil(err)

	response1 := &dlock.Response{}
	err = dlock.ReadMessage(conn1, response1, server.ConfigMaxMessage)
	assertNil(err)
	if response1.GetStatus() != dlock.ResponseStatus_Ok {
		t.Fatal("Status != Ok:", response1.GetStatus().String())
	}

	conn2, err := net.Dial("tcp", server.listeners[0].Addr().String())
	assertNil(err)
	err = conn2.SetDeadline(time.Now().Add(100 * time.Millisecond))
	assertNil(err)

	request2 := &dlock.Request{
		Type: dlock.RequestType_Lock,
		Lock: &dlock.RequestLock{
			Keys: []string{"q"},
		},
	}
	err = dlock.SendMessage(conn2, request2)
	assertNil(err)

	conn1.Close()

	response2 := &dlock.Response{}
	err = dlock.ReadMessage(conn2, response2, server.ConfigMaxMessage)
	assertNil(err)
	if response1.GetStatus() != dlock.ResponseStatus_Ok {
		t.Fatal("Status != Ok:", response1.GetStatus().String())
	}
}

func init() {
	dlock.Debug = true
	log.SetFlags(log.Flags() | log.Lmicroseconds)
}
