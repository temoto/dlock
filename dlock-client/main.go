package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

func main() {
	var (
		flagAutoKey        = flag.String("auto-key", "", "Prepend this string to full command including all arguments and use it as key. Modifies -keys.")
		flagConnect        = flag.String("connect", "", "Connect to Dlock server at this address:port")
		flagConnectTimeout = flag.Duration("connect-timeout", 10*time.Second, "Maximum time to establish TCP connection with server")
		flagExec           = flag.String("exec", "", "Command to execute")
		flagHold           = flag.Duration("hold", 0, "Hold locks at least this time even if child process finishes earlier")
		flagIdleTimeout    = flag.Duration("idle-timeout", 30*time.Second, "Maximum time to wait for beginning of server response")
		flagKeys           = flag.String("keys", "", "Keys to lock (space separated).")
		flagLockRelease    = flag.Duration("lock-release", 0, "Tell server to hold lock for exactly this time. In this mode no implicit unlocking at disconnect is performed.")
		flagLockWait       = flag.Duration("lock-wait", 0, "Lock acquire timeout")
		flagMaxMessage     = flag.Uint("max-message", 16<<10, "Maximum message length accepted by client. If server sends more - we disconnect.")
		flagReadBuffer     = flag.Uint("read-buffer", 0, "Read buffer size for sockets")
		flagReadTimeout    = flag.Duration("read-timeout", 10*time.Second, "Maximum time to receive a single message")
		flagWriteTimeout   = flag.Duration("write-timeout", 10*time.Second, "Maximum time to send a single message")
	)
	flag.Parse()

	client := NewClient(*flagConnect, *flagIdleTimeout)
	client.ConfigAutoKey = *flagAutoKey
	client.ConfigConnectTimeout = *flagConnectTimeout
	client.ConfigExec = *flagExec
	client.ConfigHold = *flagHold
	client.ConfigKeys = strings.Split(strings.TrimSpace(*flagKeys), " ")
	client.ConfigLockRelease = *flagLockRelease
	client.ConfigLockWait = *flagLockWait
	client.ConfigMaxMessage = *flagMaxMessage
	client.ConfigReadBuffer = *flagReadBuffer
	client.ConfigReadTimeout = *flagReadTimeout
	client.ConfigWriteTimeout = *flagWriteTimeout

	if len(client.ConfigAutoKey) == 0 && len(client.ConfigKeys) == 0 {
		log.Fatalln("One of -auto-key or -keys is mandatory.")
	}
	if client.ConfigExec == "" && client.ConfigHold == 0 {
		log.Fatalln("One of -exec or -hold is mandatory.")
	}

	sigIntChan := make(chan os.Signal, 1)
	signal.Notify(sigIntChan, syscall.SIGINT)
	go func() {
		<-sigIntChan
		client.Close()
	}()

	err := client.Connect()
	if err != nil {
		log.Fatalln("main: Client.Connect:", err.Error())
	}

	// client.Wait()
}
