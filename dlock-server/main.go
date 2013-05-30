package main

import (
	"flag"
	"github.com/temoto/dlock/dlock"
	"log"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"
)

func main() {
	var (
		flagBind         = flag.String("bind", "", "Bind to these address:port pairs")
		flagDebug        = flag.Bool("debug", false, "Enable debug logging")
		flagIdleTimeout  = flag.Duration("idle-timeout", 60*time.Second, "Disconnect clients without any activity within this time")
		flagReadTimeout  = flag.Duration("read-timeout", 10*time.Second, "Maximum time to receive a single message")
		flagWriteTimeout = flag.Duration("write-timeout", 10*time.Second, "Maximum time to send a single message")
		flagMaxMessage   = flag.Uint("max-message", 16<<10, "Maximum message length accepted by server. Clients trying to send more will be disconnected")
		flagReadBuffer   = flag.Uint("read-buffer", 0, "Read buffer size for sockets")
	)
	flag.Parse()

	// Set number of parallel threads to number of CPUs.
	runtime.GOMAXPROCS(runtime.NumCPU())

	dlock.Debug = *flagDebug

	server := NewServer(*flagBind, *flagIdleTimeout)
	server.ConfigDebug = *flagDebug
	server.ConfigMaxMessage = *flagMaxMessage
	server.ConfigReadBuffer = *flagReadBuffer
	server.ConfigReadTimeout = *flagReadTimeout
	server.ConfigWriteTimeout = *flagWriteTimeout

	if *flagDebug {
		log.SetFlags(log.Flags() | log.Lmicroseconds)
	}

	listenCount := server.Start()
	if listenCount == 0 {
		os.Exit(1)
	}

	sigIntChan := make(chan os.Signal, 1)
	signal.Notify(sigIntChan, syscall.SIGINT)
	go func() {
		<-sigIntChan
		if server.ConfigDebug {
			log.Printf("main: goroutines=%d", runtime.NumGoroutine())
		}
		server.Close()
	}()

	server.Wait()
}
