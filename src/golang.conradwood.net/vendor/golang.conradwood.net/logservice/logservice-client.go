package main

import (
	"flag"
	"fmt"
	"golang.conradwood.net/client"
	"golang.conradwood.net/logger"
	pb "golang.conradwood.net/logservice/proto"
	"os"
	"time"
)

// static variables for flag parser
var (
	log_status  = flag.String("status", "", "The status string to log")
	app_name    = flag.String("app_name", "", "The name of the application to log")
	repo        = flag.String("repository", "", "The name of the repository to log")
	groupname   = flag.String("groupname", "", "The name of the group to log")
	namespace   = flag.String("namespace", "", "the namespace to log")
	deplid      = flag.String("deploymentid", "", "The deployment id to log")
	sid         = flag.String("startupid", "", "The startup id to log")
	follow_flag = flag.Bool("f", false, "follow (tail -f like)")
)

func bail(err error, msg string) {
	if err == nil {
		return
	}
	fmt.Printf("%s: %s\n", msg, err)
	os.Exit(10)
}

func main() {
	flag.Parse()
	lines := flag.Args()

	if *follow_flag {
		follow()
		os.Exit(0)
	}
	queue, err := logger.NewAsyncLogQueue()
	bail(err, "Failed to create log queue")
	ad := pb.LogAppDef{
		Status:       *log_status,
		Appname:      *app_name,
		Repository:   *repo,
		Groupname:    *groupname,
		Namespace:    *namespace,
		DeploymentID: *deplid,
		StartupID:    *sid,
	}
	req := pb.LogRequest{
		AppDef: &ad,
	}
	for _, line := range lines {
		r := pb.LogLine{
			Time: time.Now().Unix(),
			Line: line,
		}
		req.Lines = append(req.Lines, &r)
		fmt.Printf("Logging: %s\n", line)
	}

	queue.LogCommandStdout(&req)
	time.Sleep(5 * time.Second)
	err = queue.Flush()
	bail(err, "Failed to send log")
	fmt.Printf("Done.\n")
}

func follow() {
	conn, err := client.DialWrapper("logservice.LogService")
	bail(err, "Failed to dial")
	defer conn.Close()
	ctx := client.SetAuthToken()

	cl := pb.NewLogServiceClient(conn)

	minlog := int64(-20)
	for {
		glr := pb.GetLogRequest{
			MinimumLogID: minlog,
		}
		lr, err := cl.GetLogCommandStdout(ctx, &glr)
		bail(err, "Failed to get Logcommandstdout")
		for _, entry := range lr.Entries {
			fmt.Printf("Log: %v\n", entry)
			if int64(entry.ID) >= minlog {
				minlog = int64(entry.ID)
			}
		}
		time.Sleep(1 * time.Second)
	}
}
