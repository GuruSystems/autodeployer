package main

// this is the main, privileged daemon. got to run as root because we're forking off
// different users from here

// fileaccess is split out to starter.go, which runs as an unprivileged user

// (this is done by virtue of exec('su',Args[0]) )
// the flag msgid goes into the startup code, so do not run the privileged daemon with that flag!

import (
	"fmt"
	"golang.conradwood.net/client"
	"golang.conradwood.net/logger"
	"google.golang.org/grpc"
	"os/signal"
	"syscall"
	//	"github.com/golang/protobuf/proto"
	"errors"
	"flag"
	pb "golang.conradwood.net/autodeployer/proto"
	lpb "golang.conradwood.net/logservice/proto"
	"golang.conradwood.net/server"
	"golang.org/x/net/context"
	"io"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

// static variables for flag parser
var (
	msgid        = flag.String("msgid", "", "A msgid indicating that we've been forked() and execing the command. used internally only")
	port         = flag.Int("port", 4000, "The server port")
	test         = flag.Bool("test", false, "set to true if you testing the server")
	deployed     []*Deployed
	portLock     = new(sync.Mutex)
	idleReaper   = flag.Int("reclaim", 5, "Reclaim terminated user accounts after `seconds`")
	startTimeout = flag.Int("start_timeout", 5, "timeout a deployment after `seconds`")
)

// information about a currently deployed application
type Deployed struct {
	// if true, then there is no application currently deployed for this user
	idle         bool
	startupMsg   string
	binary       string
	status       pb.DeploymentStatus
	ports        []int
	user         *user.User
	cmd          *exec.Cmd
	namespace    string
	groupname    string
	repo         string
	build        uint64
	exitCode     error
	url          string
	args         []string
	workingDir   string
	Stdout       io.Reader
	started      time.Time
	finished     time.Time
	lastLine     string
	downloadUser string
	downloadPW   string
	deploymentID string
	logger       *logger.AsyncLogQueue
}

// callback from the compound initialisation
func st(server *grpc.Server) error {
	s := new(AutoDeployer)
	// Register the handler object
	pb.RegisterAutoDeployerServer(server, s)
	return nil
}

func stopping() {
	fmt.Printf("Shutdown request received, slaying everyone...\n")
	slayAll()
	fmt.Printf("Shutting down, slayed everyone...\n")
	os.Exit(0)
}
func main() {
	// catch ctrl-c (for system shutdown)
	// and signal child processes
	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		stopping()
		os.Exit(0)
	}()
	defer stopping()

	flag.Parse() // parse stuff. see "var" section above
	if *msgid != "" {
		Execute()
		os.Exit(10) // should never happen
	}
	if *test {
		// testing
		go testing()
	}

	// we are brutal - if we startup we slay all deployment users
	slayAll()
	sd := server.NewServerDef()
	sd.Port = *port
	sd.Register = st
	err := server.ServerStartup(sd)
	if err != nil {
		fmt.Printf("failed to start server: %s\n", err)
	}
	fmt.Printf("Done\n")
	return

}

//*********************************************************************
func slayAll() {
	users := getListOfUsers()
	var wg sync.WaitGroup
	for _, un := range users {
		wg.Add(1)
		go func(user string) {
			defer wg.Done()
			Slay(user)
		}(un.Username)
	}
	wg.Wait()

}
func testing() {
	time.Sleep(time.Second * 1) // server needs starting up...
	ad := new(AutoDeployer)

	dp := pb.DeployRequest{
		DownloadURL: "http://localhost/application",
		Repository:  "testrepo",
		Binary:      "testapp",
		Args:        []string{"-port=${PORT1}", "-http_port=${PORT2}"},
		BuildID:     123,
	}
	dr, err := ad.Deploy(nil, &dp)
	if err != nil {
		fmt.Printf("Failed to deploy %s\n", err)
		os.Exit(10)
	}
	fmt.Printf("Deployed %v.\n", dr)
	fmt.Printf("Waiting forever...(testing a daemon)\n")
	select {}
}

/**********************************
* implementing the functions here:
***********************************/
type AutoDeployer struct {
	wtf int
}

func (s *AutoDeployer) Deploy(ctx context.Context, cr *pb.DeployRequest) (*pb.DeployResponse, error) {
	if cr.DownloadURL == "" {
		return nil, errors.New("DownloadURL is required")
	}
	if cr.Repository == "" {
		return nil, errors.New("Repositoryname is required")
	}
	fmt.Printf("Deploying %s, Build %d\n", cr.Repository, cr.BuildID)
	users := getListOfUsers()
	du := allocUser(users)
	if du == nil {
		fmt.Printf("allocUser returned no deployment entry ;(\n")
		return nil, errors.New("Failed to allocate a user. Server out of processes?")
	}
	du.started = time.Now()
	du.repo = cr.Repository
	du.build = cr.BuildID
	du.deploymentID = cr.DeploymentID
	du.namespace = cr.Namespace
	du.groupname = cr.Groupname
	_, wd := filepath.Split(du.user.HomeDir)
	wd = fmt.Sprintf("/srv/autodeployer/deployments/%s", wd)
	fmt.Printf("Deploying \"%s\" as user \"%s\" in %s\n", cr.Repository, du.user.Username, wd)
	uid, _ := strconv.Atoi(du.user.Uid)
	gid, _ := strconv.Atoi(du.user.Gid)
	err := createWorkingDirectory(wd, uid, gid)
	if err != nil {
		du.status = pb.DeploymentStatus_TERMINATED
		du.exitCode = err
		fmt.Printf("Failed to create working directory %s: %s\n", wd, err)
		return nil, err
	}
	du.startupMsg = RandomString(16)
	binname := os.Args[0]
	fmt.Printf("Binary name (self): \"%s\"\n", binname)
	if binname == "" {
		return nil, errors.New("Failed to re-exec self. check startup path of daemon")
	}
	cmd := exec.Command("su", "-s", binname, du.user.Username, "--",
		fmt.Sprintf("-token=%s", client.GetToken()),
		fmt.Sprintf("-msgid=%s", du.startupMsg))
	fmt.Printf("Executing: %v\n", cmd)
	// fill our deploystatus with stuff
	// copy deployment request to deployment descriptor
	du.downloadUser = cr.DownloadUser
	du.downloadPW = cr.DownloadPassword
	du.cmd = cmd
	du.workingDir = wd
	du.args = cr.Args
	du.url = cr.DownloadURL
	du.binary = cr.Binary

	du.status = pb.DeploymentStatus_STARTING
	du.Stdout, err = du.cmd.StdoutPipe()
	if err != nil {
		s := fmt.Sprintf("Could not get cmd output: %s\n", err)
		du.idle = true
		return nil, errors.New(s)
	}
	fmt.Printf("Starting Command: %s\n", du.toString())
	err = cmd.Start()
	if err != nil {
		fmt.Printf("Command: %v failed\n", cmd)
		du.idle = true
		return nil, err
	}
	// reap children...
	go waitForCommand(du)

	// now we need to wait for our internal startup message..
	sloop := time.Now()
	lastStatus := du.status
	for {
		if du.status != lastStatus {
			fmt.Printf("Command %s changed status from %s to %s\n", du.toString(), lastStatus, du.status)
			lastStatus = du.status
		}
		// wait
		if du.status == pb.DeploymentStatus_TERMINATED {
			if du.exitCode != nil {
				if du.lastLine == "" {
					return nil, du.exitCode
				}
				txt := fmt.Sprintf("%s (%s)", du.exitCode, du.lastLine)
				return nil, errors.New(txt)
			}
			resp := pb.DeployResponse{
				Success: true,
				Message: "OK",
				Running: false}
			return &resp, nil
		} else if du.status == pb.DeploymentStatus_EXECUSER {
			resp := pb.DeployResponse{
				Success: true,
				Message: "OK",
				Running: true}
			return &resp, nil
		}
		if time.Since(sloop) > (time.Duration(*startTimeout) * time.Second) {
			return nil, errors.New(fmt.Sprintf("Timeout after %d seconds", *startTimeout))
		}
	}
	return nil, errors.New("Deploy() in server - this codepath should never have been reached!")
}
func (s *AutoDeployer) Undeploy(ctx context.Context, cr *pb.UndeployRequest) (*pb.UndeployResponse, error) {
	if cr.ID == "" {
		return nil, errors.New("Undeployrequest requires id")
	}
	dep := entryByMsg(cr.ID)
	if dep == nil {
		return nil, errors.New(fmt.Sprintf("No deployment with id %s", cr.ID))
	}
	if cr.Block {
		fmt.Printf("Shutting down (sync): %s\n", dep.toString())
		Slay(dep.user.Username)
	} else {
		fmt.Printf("Shutting down (async): %s\n", dep.toString())
		go Slay(dep.user.Username)
	}
	res := pb.UndeployResponse{}
	return &res, nil
}

/*****************************************************
** this is called by the starter.go
** after it has forked, dropped privileges, and
** before it exec's the application
*****************************************************/
func (s *AutoDeployer) InternalStartup(ctx context.Context, cr *pb.StartupRequest) (*pb.StartupResponse, error) {

	d := entryByMsg(cr.Msgid)
	if d == nil {
		return nil, errors.New("No such deployment")
	}
	if d.status != pb.DeploymentStatus_STARTING {
		return nil, errors.New(fmt.Sprintf("Deployment in status %s not STARTING!", d.status))
	}
	d.status = pb.DeploymentStatus_DOWNLOADING

	sr := &pb.StartupResponse{
		URL:              d.url,
		Args:             d.args,
		Binary:           d.binary,
		DownloadUser:     d.downloadUser,
		DownloadPassword: d.downloadPW,
		WorkingDir:       d.workingDir,
	}
	// add some standard args (which we pass to ALL deployments)
	path := fmt.Sprintf("%s/%s/%s/%d", d.namespace, d.groupname, d.repo, d.build)
	sr.Args = append(sr.Args, fmt.Sprintf("-deployment_gurupath=%s", path))
	return sr, nil
}

func (s *AutoDeployer) Started(ctx context.Context, cr *pb.StartedRequest) (*pb.EmptyResponse, error) {
	d := entryByMsg(cr.Msgid)
	if d == nil {
		return nil, errors.New("No such deployment")
	}
	d.status = pb.DeploymentStatus_EXECUSER
	return &pb.EmptyResponse{}, nil
}
func (s *AutoDeployer) Terminated(ctx context.Context, cr *pb.TerminationRequest) (*pb.EmptyResponse, error) {
	d := entryByMsg(cr.Msgid)
	if d == nil {
		return nil, errors.New("No such deployment")
	}
	if cr.Failed {
		fmt.Printf("Child reports: %s failed.\n", d.toString())
		d.exitCode = errors.New("Unspecified OS Failure")
	} else {
		fmt.Printf("Child reports: %s exited.\n", d.toString())
	}
	d.finished = time.Now()
	d.status = pb.DeploymentStatus_TERMINATED
	return &pb.EmptyResponse{}, nil
}

// async, whenever a process exits...
func waitForCommand(du *Deployed) {
	lineOut := new(LineReader)
	buf := make([]byte, 2)
	for {
		ct, err := du.Stdout.Read(buf)
		if err != nil {
			if err != io.EOF {
				fmt.Printf("Failed to read command output: %s\n", err)
			}
			break
		}
		line := lineOut.gotBytes(buf, ct)
		if line != "" {
			checkLogger(du)
			ad := lpb.LogAppDef{
				Status:       fmt.Sprintf("%s", du.status),
				Appname:      du.binary,
				Repository:   du.repo,
				Groupname:    du.groupname,
				Namespace:    du.namespace,
				DeploymentID: du.deploymentID,
				StartupID:    du.startupMsg,
			}
			req := lpb.LogRequest{
				AppDef: &ad,
			}
			r := lpb.LogLine{
				Time: time.Now().Unix(),
				Line: line,
			}
			req.Lines = append(req.Lines, &r)
			du.logger.LogCommandStdout(&req)
			fmt.Printf(">>>>COMMAND: %s: %s\n", du.toString(), line)
			du.lastLine = line
		}
	}
	err := du.cmd.Wait()
	du.finished = time.Now()
	du.status = pb.DeploymentStatus_TERMINATED
	if du.exitCode == nil {
		du.exitCode = err
	}
	if du.exitCode != nil {
		fmt.Printf("Failed: %s (%s)\n", du.toString(), du.exitCode)
	} else {
		fmt.Printf("Exited normally: %s\n", du.toString())
	}

	Slay(du.user.Username)
	if du.logger != nil {
		du.logger.Flush()
	}
}
func Slay(username string) {
	// we clean up - to make sure we really really release resources, we "slay" the user
	cmd := exec.Command("/usr/sbin/slay", "-clean", username)
	fmt.Printf("Slaying process of user %s\n", username)
	err := cmd.Run()
	if err != nil {
		fmt.Printf("Slay failed: %s\n", err)
	} else {
		fmt.Printf("Slay %s done\n", username)
	}
}

func (s *AutoDeployer) AllocResources(ctx context.Context, cr *pb.ResourceRequest) (*pb.ResourceResponse, error) {
	res := &pb.ResourceResponse{}
	d := entryByMsg(cr.Msgid)
	if d == nil {
		return nil, errors.New("No such deployment")
	}
	d.status = pb.DeploymentStatus_RESOURCING
	fmt.Printf("Going into singleton port lock...\n")
	portLock.Lock()
	for i := 0; i < int(cr.Ports); i++ {
		res.Ports = append(res.Ports, allocPort(d))
	}
	portLock.Unlock()
	fmt.Printf("Done singleton port lock...\n")
	return res, nil
}

// we assume stuff is locked !
func allocPort(du *Deployed) int32 {
	startPort := 4100
	endPort := 7000
	for i := startPort; i < endPort; i++ {
		if !isPortInUse(i) {
			du.ports = append(du.ports, i)
			return int32(i)
		}
	}
	return 0
}
func isPortInUse(port int) bool {
	for _, d := range deployed {
		if d.idle {
			continue
		}
		for _, p := range d.ports {
			if p == port {
				return true
			}
		}

	}
	return false
}

func (s *AutoDeployer) GetDeployments(ctx context.Context, cr *pb.InfoRequest) (*pb.InfoResponse, error) {
	res := pb.InfoResponse{}
	for _, d := range deployed {
		if (d.status == pb.DeploymentStatus_TERMINATED) || (d.idle) {
			continue
		}
		dr := pb.DeployRequest{}
		dr.DownloadURL = d.url
		dr.DownloadUser = d.downloadUser
		dr.DownloadPassword = d.downloadPW
		dr.Binary = d.binary
		dr.Repository = d.repo
		dr.BuildID = d.build
		dr.DeploymentID = d.deploymentID
		dr.Args = d.args
		da := pb.DeployedApp{Deployment: &dr, ID: d.startupMsg}
		res.Apps = append(res.Apps, &da)
	}
	return &res, nil
}

/**********************************
* implementing helper functions here
***********************************/
// given a user string will get the entry for that user
// will always return one (creates one if necessary)
func entryForUser(user *user.User) *Deployed {
	for _, d := range deployed {
		if d.user.Username == user.Username {
			return d
		}
	}

	// we create a new Deployed (for a given user)
	d := &Deployed{user: user, idle: true}

	deployed = append(deployed, d)
	return d
}

// find entry by msgid. nul if none found
func entryByMsg(msgid string) *Deployed {
	for _, d := range deployed {
		if d.startupMsg == msgid {
			return d
		}
	}
	return nil
}

// given a list of users will pick one that is currently not used for deployment
// returns username
func allocUser(users []*user.User) *Deployed {
	for _, d := range deployed {
		freeEntry(d)
	}

	for _, u := range users {
		d := entryForUser(u)
		if d.idle {
			allocEntry(d)
			return d
		}
	}
	fmt.Printf("Given %d users, found NO free entry\n", len(users))
	return nil
}

// frees an entry for later usage
func freeEntry(d *Deployed) {
	// it's already idle, nothing to do
	if d.idle {
		return
	}
	// it's not idle and not terminated, so keep it!
	if d.status != pb.DeploymentStatus_TERMINATED {
		return
	}

	// it's too soon after process terminated, we keep it around for a bit
	if time.Since(d.finished) < (time.Duration(*idleReaper) * time.Second) {
		return
	}
	if d.logger != nil {
		d.logger.Flush()
		d.logger = nil
	}

	os.RemoveAll(d.workingDir)
	d.idle = true
	fmt.Printf("Reclaimed %s\n", d.toString())

}

// prepares an allocEntry for usage
func allocEntry(d *Deployed) {

	d.idle = false
	d.status = pb.DeploymentStatus_PREPARING
	checkLogger(d)
}

func checkLogger(d *Deployed) {
	if d.logger != nil {
		return
	}
	l, err := logger.NewAsyncLogQueue()
	if err != nil {
		fmt.Printf("Failed to initialize logger! %s\n", err)
	} else {
		d.logger = l
	}
}

// creates a pristine, fresh, empty, standard nice working directory
func createWorkingDirectory(dir string, uid int, gid int) error {

	// we are going to delete the entire directory, so let's make
	// sure it's the right directory!
	if !strings.HasPrefix(dir, "/srv/autodeployer") {
		return errors.New(fmt.Sprintf("%s is not absolute", dir))
	}
	err := os.RemoveAll(dir)

	if err != nil {
		return errors.New(fmt.Sprintf("Failed to remove directory %s: %s", dir, err))
	}
	err = os.Mkdir(dir, 0700)
	if err != nil {
		return errors.New(fmt.Sprintf("Failed to mkdir %s: %s", dir, err))

	}
	f, err := os.Open(dir)
	if err != nil {
		return errors.New(fmt.Sprintf("Failed to open %s: %s", dir, err))
	}
	defer f.Close()
	err = f.Chown(uid, gid)
	if err != nil {
		return errors.New(fmt.Sprintf("Failed to chown %s: %s", dir, err))
	}
	return nil
}

// cycles through users deploy1, deploy2, deploy3 ... until the first one not found
func getListOfUsers() []*user.User {
	var res []*user.User
	i := 1
	for {
		un := fmt.Sprintf("deploy%d", i)
		u, err := user.Lookup(un)
		if err != nil {
			break
		}
		res = append(res, u)
		i++
	}
	return res
}

func (d *Deployed) toString() string {
	return fmt.Sprintf("%s-%d (%s) %s", d.repo, d.build, d.startupMsg, d.status)
}
