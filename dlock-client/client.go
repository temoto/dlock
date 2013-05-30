package main

import (
	"bufio"
	"errors"
	"fmt"
	"github.com/temoto/dlock/dlock"
	"log"
	"net"
	"strings"
	"time"
)

type Client struct {
	ConfigAutoKey        string
	ConfigConnect        string
	ConfigConnectTimeout time.Duration
	ConfigDebug          bool
	ConfigExec           string
	ConfigHold           time.Duration
	ConfigIdleTimeout    time.Duration
	ConfigKeys           []string
	ConfigLockRelease    time.Duration
	ConfigLockWait       time.Duration
	ConfigMaxMessage     uint
	ConfigReadBuffer     uint
	ConfigReadTimeout    time.Duration
	ConfigWriteTimeout   time.Duration

	r       *bufio.Reader
	tcpConn *net.TCPConn
	w       *bufio.Writer
}

func NewClient(connect string, timeout time.Duration) *Client {
	return &Client{
		ConfigConnect:        connect,
		ConfigConnectTimeout: timeout,
		ConfigIdleTimeout:    timeout,
		ConfigReadTimeout:    timeout,
		ConfigWriteTimeout:   timeout,
	}
}

func (c *Client) Close(linger int) (err error) {
	if err = c.tcpConn.SetLinger(linger); err != nil {
		return err
	}
	return c.tcpConn.Close()
}

func (c *Client) Connect() error {
	defer c.profileTime("Client.Connect", time.Now())
	conn, err := net.DialTimeout("tcp", c.ConfigConnect, c.ConfigConnectTimeout)
	if err != nil {
		return err
	}
	var ok bool
	if c.tcpConn, ok = conn.(*net.TCPConn); !ok {
		return errors.New("Client.Connect: failed to cast net.Conn to net.TCPConn")
	}

	if err = c.tcpConn.SetLinger(0); err != nil {
		return err
	}

	if c.ConfigReadBuffer == 0 {
		c.r = bufio.NewReader(c.tcpConn)
	} else {
		if err = c.tcpConn.SetReadBuffer(int(c.ConfigReadBuffer)); err != nil {
			return err
		}
		c.r = bufio.NewReaderSize(c.tcpConn, int(c.ConfigReadBuffer))
	}
	c.w = bufio.NewWriter(c.tcpConn)
	return nil
}

func (c *Client) Lock(keys []string, wait, release time.Duration) (err error) {
	defer c.profileTime("Client.Lock", time.Now())

	if wait != 0 {
		ch := make(chan error, 1)
		go func() { ch <- c.lock(keys, wait, release) }()
		select {
		case err = <-ch:
		case <-time.After(wait):
			err = dlock.ErrorLockAcquireTimeout
			c.Close(0)
		}
	} else {
		err = c.lock(keys, wait, release)
	}
	return err
}

func (c *Client) lock(keys []string, wait, release time.Duration) (err error) {
	if c.tcpConn == nil {
		if err = c.Connect(); err != nil {
			return err
		}
	}

	request := &dlock.Request{
		Type: dlock.RequestType_Lock.Enum(),
		Lock: &dlock.RequestLock{
			Keys:         keys,
			WaitMicro:    new(uint64),
			ReleaseMicro: new(uint64),
		},
	}
	*request.Lock.WaitMicro = uint64(wait / time.Microsecond)
	*request.Lock.ReleaseMicro = uint64(release / time.Microsecond)
	if err = dlock.SendMessage(c.w, request); err != nil {
		return err
	}
	if err = c.w.Flush(); err != nil {
		return err
	}

	response := &dlock.Response{}
	if err = dlock.ReadMessage(c.r, response, c.ConfigMaxMessage); err != nil {
		return err
	}
	if response.GetStatus() != dlock.ResponseStatus_Ok {
		return errors.New(fmt.Sprintf("Remote error locking keys %v: %s %s failed keys: %v",
			keys, response.GetStatus().String(), response.GetErrorText(), response.Keys))
	}

	return nil
}

func (c *Client) Ping() (err error) {
	defer c.profileTime("Client.Ping", time.Now())
	if c.tcpConn == nil {
		if err = c.Connect(); err != nil {
			return err
		}
	}

	request := &dlock.Request{
		Type: dlock.RequestType_Ping.Enum(),
	}
	if err = dlock.SendMessage(c.w, request); err != nil {
		return err
	}
	if err = c.w.Flush(); err != nil {
		return err
	}

	response := &dlock.Response{}
	if err = dlock.ReadMessage(c.r, response, c.ConfigMaxMessage); err != nil {
		return err
	}
	if response.GetStatus() != dlock.ResponseStatus_Ok {
		return errors.New(fmt.Sprintf("Ping: Remote error: %s %s",
			response.GetStatus().String(), response.GetErrorText()))
	}

	return nil
}

func (c *Client) profileTime(tag string, t1 time.Time) {
	d := time.Now().Sub(t1)
	if c.ConfigDebug {
		log.Printf("%s time=%s", tag, d)
	}
}

func (c *Client) parseKeys(in string) []string {
	out := make([]string, 0, len(in)/10)
	for _, key := range strings.Split(in, " ") {
		key = strings.TrimSpace(key)
		if len(key) > 0 {
			out = append(out, key)
		}
	}
	return out
}
