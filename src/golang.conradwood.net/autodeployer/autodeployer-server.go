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
	rpb "golang.conradwood.net/registrar/proto"
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
	machineGroup = flag.String("machinegroup", "worker", "the group a specific machine is in")
	testfile     = flag.String("cfgfile", "", "config file (for testing)")
)

// information about a currently deployed application
type Deployed struct {
	// if true, then there is no application currently deployed for this user
	idle             bool
	startupMsg       string
	binary           string
	status           pb.DeploymentStatus
	ports            []int
	user             *user.User
	cmd              *exec.Cmd
	namespace        string
	groupname        string
	repo             string
	build            uint64
	exitCode         error
	url              string
	args             []string
	workingDir       string
	Stdout           io.Reader
	started          time.Time
	finished         time.Time
	lastLine         string
	downloadUser     string
	downloadPW       string
	deploymentID     string
	deploymentpath   string
	logger           *logger.AsyncLogQueue
	autoRegistration []*pb.AutoRegistration
}

func isTestMode() bool {
	if *testfile != "" {
		return true
	}
	return false
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
	flag.Parse() // parse stuff. see "var" section above
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

	if *msgid != "" {
		Execute()
		os.Exit(10) // should never happen
	}
	if *test {
		// testing
		go testing()
		return
	}

	// we are brutal - if we startup we slay all deployment users
	slayAll()
	if *testfile != "" {
		go ApplyTestFile()
		fmt.Printf("Apply testfile: done\n")
	}

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
	if isTestMode() {
		fmt.Printf("Not slaying - testmode activated\n")
		return
	}
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
* implementing test functions here
***********************************/
func ApplyTestFile() {
	time.Sleep(2 * time.Second)
	fmt.Printf("Applying testfile: %s\n", *testfile)
	gd, err := ParseFile(*testfile)
	if err != nil {
		fmt.Printf("failed to parse file %s: %s\n", *testfile, err)
		return
	}

	ad := new(AutoDeployer)

	for _, x := range gd.Groups {
		if x.Namespace == "" {
			x.Namespace = gd.Namespace
		}
	}
	for _, g := range gd.Groups {
		for _, app := range g.Applications {
			fmt.Printf("Deploying application %s from %s with %d autoregistration ports\n", app.Binary, app.DownloadURL, len(app.AutoRegs))
			dp := pb.DeployRequest{
				DownloadURL:      app.DownloadURL,
				DownloadUser:     app.DownloadUser,
				DownloadPassword: app.DownloadPassword,
				Binary:           app.Binary,
				Args:             []string{"-port=${PORT1}", "-http_port=${PORT2}"},
				Repository:       app.Repository,
				BuildID:          app.BuildID,
				DeploymentID:     "testdeplid",
				Namespace:        "namespace",
				Groupname:        "groupname",
				AutoRegistration: app.AutoRegs,
			}
			_, err := ad.Deploy(nil, &dp)
			if err != nil {
				fmt.Printf("Deployment error: %s\n", err)
				return
			}
		}
	}
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
	for _, ar := range cr.AutoRegistration {
		_, err := convStringToApitypes(ar.ApiTypes)
		if err != nil {
			return nil, errors.New(fmt.Sprintf("Failed to convert apitypes for %v: %s", ar, err))
		}
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
	du.autoRegistration = cr.AutoRegistration
	du.downloadUser = cr.DownloadUser
	du.downloadPW = cr.DownloadPassword
	du.args = cr.Args
	du.url = cr.DownloadURL
	du.binary = cr.Binary
	path := fmt.Sprintf("%s/%s/%s/%d", du.namespace, du.groupname, du.repo, du.build)
	du.deploymentpath = path

	_, wd := filepath.Split(du.user.HomeDir)
	wd = fmt.Sprintf("/srv/autodeployer/deployments/%s", wd)
	fmt.Printf("Deploying \"%s\" as user \"%s\" in %s\n", cr.Repository, du.user.Username, wd)
	uid, _ := strconv.Atoi(du.user.Uid)
	gid, _ := strconv.Atoi(du.user.Gid)
	err := createWorkingDirectory(wd, uid, gid)
	if err != nil && (!isTestMode()) {
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

	du.cmd = cmd
	du.workingDir = wd

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

	sr.Args = append(sr.Args, fmt.Sprintf("-deployment_gurupath=%s", d.deploymentpath))
	return sr, nil
}

// triggered by the unprivileged startup code
func (s *AutoDeployer) Started(ctx context.Context, cr *pb.StartedRequest) (*pb.EmptyResponse, error) {
	du := entryByMsg(cr.Msgid)
	if du == nil {
		return nil, errors.New("No such deployment")
	}
	du.status = pb.DeploymentStatus_EXECUSER
	du.StartupCodeExec()
	return &pb.EmptyResponse{}, nil
}

// triggered by the unprivileged startup code
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

	// here we end up when our command terminates. it's still the privileged
	// server
	du.StartupCodeFinished(err)

}
func Slay(username string) {
	// we clean up - to make sure we really really release resources, we "slay" the user
	cmd := exec.Command("/usr/sbin/slay", "-clean", username)
	//fmt.Printf("Slaying process of user %s\n", username)
	err := cmd.Run()
	if err != nil {
		fmt.Printf("Slay user %s failed: %s\n", username, err)
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

func (s *AutoDeployer) GetMachineInfo(ctx context.Context, cr *pb.MachineInfoRequest) (*pb.MachineInfoResponse, error) {
	res := pb.MachineInfoResponse{
		MachineGroup: *machineGroup,
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
			fmt.Printf("Max users: %d (ended with %s)\n", i, err)
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

// called by the main thread, once the startup code claims it handed control
// to the program
func (du *Deployed) StartupCodeExec() {
	// auto register stuff
	fmt.Printf("Got %d autoregistration services to take care of\n", len(du.autoRegistration))
	for _, ar := range du.autoRegistration {
		port := du.getPortByName(ar.Portdef)
		if port == 0 {
			fmt.Printf("Broken autoregistration (%v) - no portdef\n", ar)
			continue
		}
		apiTypes, err := convStringToApitypes(ar.ApiTypes)
		if err != nil {
			fmt.Printf("Broken autoregistration (%v) - api error %s\n", ar, err)
			continue
		}
		for _, at := range apiTypes {
			fmt.Printf("Autoregistering %s on port %d, type=%v\n", ar.ServiceName, port, at)
			if at == rpb.Apitype_tcp {
				sd := server.NewTCPServerDef(ar.ServiceName)
				sd.Port = port
				sd.DeployPath = du.deploymentpath
				server.AddRegistry(sd)
			} else if at == rpb.Apitype_html {
				sd := server.NewHTMLServerDef(ar.ServiceName)
				sd.Port = port
				sd.DeployPath = du.deploymentpath
				server.AddRegistry(sd)
			} else {
				fmt.Printf("Cannot (yet) auto-register apitype: %s\n", at)
				continue
			}

		}
	}
}

// called by the main thread (privileged) when our forked startup.go finished
func (du *Deployed) StartupCodeFinished(exitCode error) {
	// unregister the ports...
	err := server.UnregisterPortRegistry(du.ports)
	if err != nil {
		fmt.Printf("Failed to unregister port %s\n", err)
	}
	du.finished = time.Now()
	du.status = pb.DeploymentStatus_TERMINATED
	if du.exitCode == nil {
		du.exitCode = exitCode
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
func (du *Deployed) getPortByName(name string) int {
	if !strings.HasPrefix(name, "${PORT") {
		fmt.Printf("Invalid port name: %s\n", name)
		return 0
	}
	psn := name[6 : len(name)-1]
	pn, err := strconv.Atoi(psn)
	if err != nil {
		fmt.Printf("Could not convert port by name %s to portnumber: %s\n", name, err)
		return 0
	}
	if pn >= len(du.ports) {
		fmt.Printf("Port %d not allocated (%d)\n", pn, len(du.ports))
		return 0
	}
	po := du.ports[pn]
	fmt.Printf("Port %s == %d\n", psn, po)
	return po
}
func convStringToApitypes(apitypestring string) ([]rpb.Apitype, error) {
	var res []rpb.Apitype
	asa := strings.Split(apitypestring, ",")
	for _, as := range asa {
		as = strings.TrimLeft(as, " ")
		as = strings.TrimRight(as, " ")
		fmt.Printf("Converting: \"%s\"\n", as)
		v, ok := rpb.Apitype_value[as]
		if !ok {
			return nil, errors.New(fmt.Sprintf("unknown apitype \"%s\"", as))
		}
		av := rpb.Apitype(v)
		res = append(res, av)
	}
	return res, nil
}
