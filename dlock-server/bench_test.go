package main

import (
	"github.com/temoto/dlock/dlock"
	"log"
	"net"
	"testing"
	"time"
)

func BenchmarkLockEmpty(b *testing.B) {
	server := NewServer(":0", 1*time.Second)
	if n := server.Start(); n != 1 {
		b.Fatal("NewServer: expected 1 listener, Server.Start():", n)
	}
	conn1, err := net.Dial("tcp", server.listeners[0].Addr().String())
	assertNil(err)
	// assertNil(conn1.SetDeadline(time.Now().Add(100 * time.Millisecond)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		assertNil(dlock.SendMessage(conn1, &dlock.Request{
			Version: 2,
			Type:    dlock.RequestType_Lock,
			Lock: &dlock.RequestLock{
				Keys: []string{"q"},
			},
		}))
		response := &dlock.Response{}
		assertNil(dlock.ReadMessage(conn1, response, 100))
		assertNil(dlock.SendMessage(conn1, &dlock.Request{
			Version: 2,
			Type:    dlock.RequestType_Unlock,
			Lock: &dlock.RequestLock{
				Keys: []string{"q"},
			},
		}))
		response = &dlock.Response{}
		assertNil(dlock.ReadMessage(conn1, response, 100))
	}
	conn1.Close()
	server.Close()
	server.Wait()
}

type Null struct{}

func (Null) Write(b []byte) (int, error) {
	return len(b), nil
}

func init() {
	dlock.Debug = false
	log.SetOutput(Null{})
}
