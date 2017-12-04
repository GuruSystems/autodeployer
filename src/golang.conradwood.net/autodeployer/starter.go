package main

import (
	"errors"
	"fmt"
	pb "golang.conradwood.net/autodeployer/proto"
	"golang.conradwood.net/client"
	"google.golang.org/grpc"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"
)

// this is the non-privileged section of the autodeployer

//*********************************************************************
// execute whatever passed in as msgid and never returns
// (exits if childprocess exits)
// this is the 2nd part of the server (execed by main, this part is running unprivileged)
func Execute() {
	// redirect stderr to stdout (to capture panics)
	syscall.Dup2(int(os.Stdout.Fd()), int(os.Stderr.Fd()))

	// we're speaking to the local server only ever
	serverAddr := fmt.Sprintf("localhost:%d", *port)
	creds := client.GetClientCreds()
	fmt.Printf("Connecting to local autodeploy server:%s...\n", serverAddr)
	conn, err := grpc.Dial(serverAddr, grpc.WithTransportCredentials(creds))
	if err != nil {
		fmt.Println("fail to dial: %v", err)
		return
	}
	defer conn.Close()
	fmt.Println("Creating client...")
	cl := pb.NewAutoDeployerClient(conn)
	ctx := client.SetAuthToken()

	// the the server we're starting to deploy and get the parameters for deployment
	sr := pb.StartupRequest{Msgid: *msgid}
	srp, err := cl.InternalStartup(ctx, &sr)
	if err != nil {
		fmt.Printf("Failed to startup: %s\n", err)
		os.Exit(10)
	}
	if srp.URL == "" {
		fmt.Printf("no download url in startup response\n")
		os.Exit(10)
	}

	// change to my working directory
	err = os.Chdir(srp.WorkingDir)
	if err != nil {
		fmt.Printf("Failed to Chdir() to %s: %s\n", srp.WorkingDir, err)
	}
	fmt.Printf("Chdir() to %s\n", srp.WorkingDir)
	// download the binary and/or archive
	binary := "executable"
	if srp.Binary != "" {
		binary = srp.Binary
	}
	fmt.Printf("Downloading binary from %s\n", srp.URL)
	err = downloadBinary(srp.URL, binary, srp.DownloadUser, srp.DownloadPassword)
	if err != nil {
		fmt.Printf("Failed to download from %s: %s\n", srp.URL, err)
		os.Exit(10)
	}

	// execute the binary
	ports := countPortCommands(srp.Args)

	fmt.Printf("Getting resources\n")
	resources, err := cl.AllocResources(ctx, &pb.ResourceRequest{Msgid: *msgid, Ports: int32(ports)})
	if err != nil {
		fmt.Printf("Failed to alloc resources: %s\n", err)
		os.Exit(10)
	}
	fmt.Printf("Start commandline: %s %v (%d ports)\n", binary, srp.Args, ports)
	rArgs := replacePorts(srp.Args, resources.Ports)
	fmt.Printf("Starting binary \"%s\" with %d args:\n", binary, len(srp.Args))
	for _, s := range rArgs {
		fmt.Printf("Arg: \"%s\"\n", s)
	}
	path := "./"
	fullb := fmt.Sprintf("%s/%s", path, binary)
	err = os.Chmod(fullb, 0500)
	if err != nil {
		fmt.Printf("Failed to chmod %s: %s\n", fullb, err)
		os.Exit(10)
	}

	fmt.Printf("Starting user application..\n")
	cmd := exec.Command(fullb, rArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Start()
	if err != nil {
		fmt.Printf("Failed to start(): %s\n", err)
		os.Exit(10)
	}
	_, err = cl.Started(ctx, &pb.StartedRequest{Msgid: *msgid})
	if err != nil {
		fmt.Printf("Failed to inform daemon about pending startup. aborting. (%s)\n", err)
		os.Exit(10)
	}
	err = cmd.Wait()
	if err == nil {
		fmt.Printf("Command completed with no error\n")
	} else {
		fmt.Printf("Command completed: %s\n", err)
	}
	failed := err != nil
	cl.Terminated(ctx, &pb.TerminationRequest{Msgid: *msgid, Failed: failed})
	os.Exit(0)
}

//*********************************************************************
// replace ${PORTx} with actual ports
func replacePorts(args []string, ports []int32) []string {
	res := []string{}
	for _, r := range args {
		n := r
		for i := 0; i < len(ports); i++ {
			s := fmt.Sprintf("${PORT%d}", i+1)
			n = strings.Replace(n, s, fmt.Sprintf("%d", ports[i]), -1)
		}
		res = append(res, n)
	}
	return res
}

// how many ${PORTx} variables do we have in our string?
func countPortCommands(args []string) int {
	res := 0
	for i := 1; i < 20; i++ {
		s := fmt.Sprintf("${PORT%d}", i)
		for _, r := range args {
			if strings.Contains(r, s) {
				res = res + 1
			}
		}
	}
	return res
}

// download a file and depending on its type extract the archive
func downloadBinary(url string, target string, user string, pw string) error {
	err := downloadFromURL(url, target, user, pw)
	return err
}

// download a file to 'target'
func downloadFromURL(url string, target string, user string, pw string) error {
	fileName := target
	fmt.Println("Downloading", url, "to", fileName)

	// TODO: check file existence first with io.IsExist
	output, err := os.Create(fileName)
	if err != nil {
		fmt.Println("Error while creating", fileName, "-", err)
		return err
	}
	defer output.Close()
	tr := &http.Transport{
		MaxIdleConns:       10,
		IdleConnTimeout:    30 * time.Second,
		DisableCompression: true,
	}
	client := &http.Client{
		Transport: tr,
	}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		fmt.Println("Error while downloading", url, "-", err)
		return err
	}
	if user != "" {
		fmt.Printf("Setting http username for url %s to \"%s\"\n", url, user)
		req.SetBasicAuth(user, pw)
	}
	response, err := client.Do(req)
	if err != nil {
		fmt.Println("Error while downloading", url, "-", err)
		return err
	}
	defer response.Body.Close()
	fmt.Printf("Http.Get() response code: %d (code is not necessarily an error)\n", response.StatusCode)
	if isHttpError(response.StatusCode) {
		s := fmt.Sprintf("http error: %d (%s)", response.StatusCode, response.Status)
		fmt.Println(s)
		return errors.New(s)
	}
	n, err := io.Copy(output, response.Body)
	if err != nil {
		fmt.Println("Error while downloading", url, "-", err)
		return err
	}

	fmt.Println(n, "bytes downloaded.")
	return nil
}

func isHttpError(code int) bool {
	if (code >= 400) && (code <= 499) {
		return true
	}
	if (code >= 500) && (code <= 599) {
		return true
	}
	return false
}
