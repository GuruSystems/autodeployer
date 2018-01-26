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
	sent    bool
	created int64
	line    string
}
type AsyncLogQueue struct {
	lock           sync.Mutex
	entries        []*QueueEntry
	lastErrPrinted time.Time
	appDef         *pb.LogAppDef
}

func NewAsyncLogQueue(appname, repo, group, namespace, deplid string) (*AsyncLogQueue, error) {
	b := &pb.LogAppDef{Appname: appname,
		Repository:   repo,
		Groupname:    group,
		Namespace:    namespace,
		DeploymentID: deplid,
	}
	alq := &AsyncLogQueue{appDef: b}
	t := time.NewTicker(2 * time.Second)
	go func(a *AsyncLogQueue) {
		for _ = range t.C {
			a.Flush()
		}
	}(alq)
	return alq, nil
}
func (alq *AsyncLogQueue) LogCommandStdout(line string, status string) error {
	alq.lock.Lock()
	defer alq.lock.Unlock()
	qe := QueueEntry{sent: false,
		created: time.Now().Unix(),
		line:    line}
	alq.lock.Lock()
	alq.entries = append(alq.entries, &qe)
	alq.lock.Unlock()
	return nil
}

func (alq *AsyncLogQueue) Flush() error {
	alq.lock.Lock()
	defer alq.lock.Unlock()
	// fmt.Printf("Sending %d log entries\n", len(alq.entries))
	if len(alq.entries) == 0 {
		// save ourselves from dialing and stuff
		return nil
	}
	lr := pb.LogRequest{
		AppDef: alq.appDef,
	}
	for {
		conn, err := client.DialWrapper("logservice.LogService")
		if err != nil {
			return errors.New(fmt.Sprintf("Logqueue flush error: %s", err))
		}
		defer conn.Close()
		ctx := client.SetAuthToken()
		cl := pb.NewLogServiceClient(conn)

		for _, qe := range alq.entries {
			lr.Lines = append(lr.Lines, &pb.LogLine{Time: qe.created, Line: qe.line})
		}
		_, err = cl.LogCommandStdout(ctx, &lr)
		if err != nil {
			if time.Since(alq.lastErrPrinted) > (10 * time.Second) {
				fmt.Printf("Failed to send log: %s\n", err)
				alq.lastErrPrinted = time.Now()
			}
		}
	}
	// all Done
	// so clear the array so we free up the memory
	alq.entries = alq.entries[:0]
	return nil

}
