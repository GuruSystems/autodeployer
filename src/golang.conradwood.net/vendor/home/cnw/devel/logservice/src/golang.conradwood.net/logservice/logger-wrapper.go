package main

import (
	"errors"
	"flag"
	"fmt"
	"golang.conradwood.net/logger"
	lpb "golang.conradwood.net/logservice/proto"
	"io"
	"os"
	"os/exec"
	"syscall"
	"time"
)

// static variables for flag parser
var (
	log_status = flag.String("status", "", "The status string to log")
	app_name   = flag.String("appname", "", "The name of the application to log")
	repo       = flag.String("repository", "", "The name of the repository to log")
	groupname  = flag.String("groupname", "", "The name of the group to log")
	namespace  = flag.String("namespace", "", "the namespace to log")
	deplid     = flag.String("deploymentid", "", "The deployment id to log")
	sid        = flag.String("startupid", "", "The startup id to log")
)

type com struct {
	Cmd    *exec.Cmd
	Name   string
	Paras  []string
	Stdout io.ReadCloser
	code   int
	logger *logger.AsyncLogQueue
}

func (c *com) toString() string {
	return c.Name
}

func bail(err error, msg string) {
	if err == nil {
		return
	}
	fmt.Printf("%s: %s\n", msg, err)
	os.Exit(10)
}

func main() {
	flag.Parse()
	// redirect stderr to stdout (to capture panics)
	syscall.Dup2(int(os.Stdout.Fd()), int(os.Stderr.Fd()))

	paras := flag.Args()
	name := paras[0]
	paras = paras[1:]
	cmd := com{Name: name, Paras: paras}
	err := run(&cmd)
	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(10)
	}
	os.Exit(0)
}
func run(cmd *com) error {
	var err error
	c := exec.Command(cmd.Name, cmd.Paras...)
	cmd.Cmd = c
	cmd.Stdout, err = cmd.Cmd.StdoutPipe()
	if err != nil {
		s := fmt.Sprintf("Could not get cmd output: %s\n", err)
		return errors.New(s)
	}
	fmt.Printf("Starting Command: %s\n", cmd.Name)
	err = cmd.Cmd.Start()
	if err != nil {
		fmt.Printf("Command: %v failed\n", cmd)
		return err
	}
	// reap children...
	err = waitForCommand(cmd)
	return err
}

// async, whenever a process exits...
func waitForCommand(cmd *com) error {

	lineOut := new(LineReader)
	buf := make([]byte, 2)
	for {
		ct, err := cmd.Stdout.Read(buf)
		if err != nil {
			if err != io.EOF {
				fmt.Printf("Failed to read command output: %s\n", err)
			}
			break
		}
		line := lineOut.gotBytes(buf, ct)
		if line != "" {
			cmd.checkLogger()
			ad := lpb.LogAppDef{
				Status:       "EXECUSER",
				Appname:      *app_name,
				Repository:   *repo,
				Groupname:    *groupname,
				Namespace:    *namespace,
				DeploymentID: *deplid,
				StartupID:    *sid,
			}
			req := lpb.LogRequest{
				AppDef: &ad,
			}
			r := lpb.LogLine{
				Time: time.Now().Unix(),
				Line: line,
			}
			fmt.Printf("Logging: \"%s\"\n", line)
			req.Lines = append(req.Lines, &r)
			cmd.logger.LogCommandStdout(&req)
		}
	}
	err := cmd.Cmd.Wait()
	return err

}
func (c *com) checkLogger() {
	if c.logger != nil {
		return
	}
	l, err := logger.NewAsyncLogQueue()
	if err != nil {
		fmt.Printf("Failed to initialize logger! %s\n", err)
	} else {
		c.logger = l
	}
}
