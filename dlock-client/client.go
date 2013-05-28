package main

import (
	"bufio"
	"errors"
	"fmt"
	"github.com/temoto/dlock/dlock"
	"net"
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

func (c *Client) Close() {
	c.tcpConn.Close()
}

func (c *Client) Connect() error {
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
