package main

import (
	"flag"
	"github.com/temoto/dlock/dlock"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"sync"
	"syscall"
	"time"
)

func main() {
	var (
		flagAutoKey        = flag.String("auto-key", "", "Prepend this string to full command including all arguments and use it as key. Auto key is appended to -keys.")
		flagConnect        = flag.String("connect", "", "Connect to Dlock server at this address:port")
		flagConnectTimeout = flag.Duration("connect-timeout", 10*time.Second, "Maximum time to establish TCP connection with server")
		flagDebug          = flag.Bool("debug", false, "Debug logging")
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

	// Set number of parallel threads to number of CPUs.
	runtime.GOMAXPROCS(runtime.NumCPU())

	dlock.Debug = *flagDebug

	client := NewClient(*flagConnect, *flagIdleTimeout)
	client.ConfigAutoKey = *flagAutoKey
	client.ConfigConnectTimeout = *flagConnectTimeout
	client.ConfigExec = *flagExec
	client.ConfigDebug = *flagDebug
	client.ConfigHold = *flagHold
	client.ConfigKeys = client.parseKeys(*flagKeys)
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
		client.Close(0)
	}()

	err := client.Connect()
	if err != nil {
		log.Fatalln("main: Client.Connect:", err.Error())
	}

	err = client.Lock(client.ConfigKeys, client.ConfigLockWait, *maxDuration(&client.ConfigHold, &client.ConfigLockRelease))
	if err != nil {
		log.Fatalln("main: Client.Lock:", err.Error())
	}

	exitCode := 0
	stopWait := sync.WaitGroup{}

	if client.ConfigHold != 0 {
		stopWait.Add(1)
		go func() {
			time.Sleep(client.ConfigHold)
			stopWait.Done()
		}()
	}
	if client.ConfigExec != "" {
		stopWait.Add(1)
		go func() {
			defer stopWait.Done()
			aname, err := exec.LookPath("sh")
			if err != nil {
				log.Fatalln("sh is required to run -exec program. Error:", err.Error())
			}
			attr := &os.ProcAttr{}
			p, err := os.StartProcess(aname, []string{client.ConfigExec}, attr)
			if err != nil {
				log.Fatalln(err.Error())
			}
			state, err := p.Wait()
			if err != nil {
				log.Fatalln(err.Error())
			}
			if sysStatus, ok := state.Sys().(syscall.WaitStatus); ok {
				exitCode = sysStatus.ExitStatus()
			}
		}()
	}

	stopWait.Wait()

	// client.Wait()
	os.Exit(exitCode)
}

func maxDuration(d1, d2 *time.Duration) *time.Duration {
	if *d1 >= *d2 {
		return d1
	}
	return d2
}
