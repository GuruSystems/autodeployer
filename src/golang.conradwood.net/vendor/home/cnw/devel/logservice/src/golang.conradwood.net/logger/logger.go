package logger

import (
	"errors"
	"fmt"
	"golang.conradwood.net/client"
	pb "golang.conradwood.net/logservice/proto"
	"sync"
	"time"
)

type QueueEntry struct {
	sent bool
	line string
}
type AsyncLogQueue struct {
	lock           sync.Mutex
	entries        []*QueueEntry
	lastErrPrinted time.Time
	appDef         *pb.LogAppDef
}

func NewAsyncLogQueue(b *pb.LogRequest) (*AsyncLogQueue, error) {
	alq := &AsyncLogQueue{logRequest: b}
	t := time.NewTicker(2 * time.Second)
	go func(a *AsyncLogQueue) {
		for _ = range t.C {
			a.Flush()
		}
	}(alq)
	return alq, nil
}
func (alq *AsyncLogQueue) LogCommandStdout(line string, status string) error {
	qe := QueueEntry{sent: false,
		line: lr}
	alq.lock.Lock()
	alq.entries = append(alq.entries, &qe)
	alq.lock.Unlock()
	return nil
}

func (alq *AsyncLogQueue) Flush() error {
	var lasterr error
	alq.lock.Lock()
	defer alq.lock.Unlock()
	// fmt.Printf("Sending %d log entries\n", len(alq.entries))
	if len(alq.entries) == 0 {
		// save ourselves from dialing and stuff
		return nil
	}
	retries := 5
	for {
		conn, err := client.DialWrapper("logservice.LogService")
		if err != nil {
			return errors.New(fmt.Sprintf("Logqueue flush error: %s", err))
		}
		defer conn.Close()
		ctx := client.SetAuthToken()
		cl := pb.NewLogServiceClient(conn)

		lasterr = nil
		for _, qe := range alq.entries {
			if qe.sent {
				continue
			}
			_, err := cl.LogCommandStdout(ctx, qe.logRequest)
			if err != nil {
				if time.Since(alq.lastErrPrinted) > (10 * time.Second) {
					fmt.Printf("Failed to send log: %s\n", err)
					alq.lastErrPrinted = time.Now()
				}
				lasterr = err
			} else {
				qe.sent = true
			}
		}
		if lasterr == nil {
			break
		}
		retries--
		if retries == 0 {
			return errors.New(fmt.Sprintf("Failed to send logs. last error: %s", lasterr))
		}
	}
	// all Done
	// so clear the array so we free up the memory
	alq.entries = alq.entries[:0]
	return nil

}
