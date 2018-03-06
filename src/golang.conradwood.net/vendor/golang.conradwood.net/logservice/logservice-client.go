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
	match       = flag.String("filter", "", "fuzzy magic match filter to apply to logentries' application")
	app_name    = flag.String("appname", "", "The name of the application to log or to filter on")
	repo        = flag.String("repository", "", "The name of the repository to log")
	groupname   = flag.String("groupname", "", "The name of the group to log or to filter on")
	namespace   = flag.String("namespace", "", "the namespace to log or to filter on")
	deplid      = flag.String("deploymentid", "", "The deployment id to log")
	sid         = flag.String("startupid", "", "The startup id to log or to filter on")
	follow_flag = flag.Bool("f", false, "follow (tail -f like)")
	maxLines    = flag.Int("n", 500, "Maximum lines to retrieve")
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
	if len(lines) == 0 {
		showLog()
		os.Exit(0)
	}
	queue, err := logger.NewAsyncLogQueue(*app_name, *repo, *groupname, *namespace, *deplid)
	bail(err, "Failed to create log queue")
	for _, line := range lines {
		queue.LogCommandStdout(line, "EXECUSER")
		fmt.Printf("Logging: %s\n", line)
	}

	time.Sleep(5 * time.Second)
	err = queue.Flush()
	bail(err, "Failed to send log")
	fmt.Printf("Done.\n")
}

func showLog() {
	conn, err := client.DialWrapper("logservice.LogService")
	bail(err, "Failed to dial")
	defer conn.Close()
	ctx := client.SetAuthToken()

	cl := pb.NewLogServiceClient(conn)

	minlog := int64(0 - *maxLines)
	glr := pb.GetLogRequest{
		MinimumLogID: minlog,
	}
	addFilters(&glr)

	lr, err := cl.GetLogCommandStdout(ctx, &glr)
	if err != nil {
		fmt.Printf("Getting available apps...\n")
		x, xe := cl.GetApps(ctx, &pb.EmptyRequest{})
		if xe == nil {
			for _, ld := range x.AppDef {
				fmt.Printf("Application: \"%s\", Namespace: \"%s\", Repository: \"%s\", Groupname: \"%s\"\n", ld.Appname, ld.Namespace, ld.Repository, ld.Groupname)
			}
		}
		bail(err, "Failed to get Logcommandstdout")
	}
	for _, entry := range lr.Entries {
		printLogEntry(entry)
		if int64(entry.ID) >= minlog {
			minlog = int64(entry.ID)
		}
	}
	time.Sleep(1 * time.Second)
}

func follow() {
	conn, err := client.DialWrapper("logservice.LogService")
	bail(err, "Failed to dial")
	defer conn.Close()

	cl := pb.NewLogServiceClient(conn)

	minlog := int64(-20)
	i := 0
	for {
		glr := pb.GetLogRequest{
			MinimumLogID: minlog,
		}
		addFilters(&glr)
		fmt.Printf("Querying %d...\r", i)
		i++
		ctx := client.SetAuthToken()
		lr, err := cl.GetLogCommandStdout(ctx, &glr)
		bail(err, "Failed to get Logcommandstdout")
		for _, entry := range lr.Entries {
			printLogEntry(entry)
			if int64(entry.ID) >= minlog {
				minlog = int64(entry.ID)
			}
		}
		time.Sleep(1 * time.Second)
	}
}

func printLogEntry(e *pb.LogEntry) {
	t := time.Unix(int64(e.Occured), 0)
	ts := t.String()
	//	fmt.Printf("le: %v\n", e)
	//	return
	fmt.Printf("%s %d %s %s repo:%s group:%s app:%s: %s\n", ts, e.ID, e.Host, e.AppDef.Status,
		e.AppDef.Repository, e.AppDef.Groupname, e.AppDef.Appname,
		e.Line)
}

func addFilters(glr *pb.GetLogRequest) {
	lf := &pb.LogFilter{
		FuzzyMatch: *match,
	}
	la := &pb.LogAppDef{
		Appname:    *app_name,
		Groupname:  *groupname,
		Namespace:  *namespace,
		StartupID:  *sid,
		Repository: *repo,
	}
	lf.AppDef = la
	glr.LogFilter = append(glr.LogFilter, lf)
}
