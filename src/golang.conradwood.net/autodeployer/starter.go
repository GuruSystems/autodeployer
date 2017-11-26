package main

import (
	"fmt"
	pb "golang.conradwood.net/autodeployer/proto"
	"golang.conradwood.net/client"
	"google.golang.org/grpc"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
)

// this is the non-privileged section of the autodeployer

//*********************************************************************
// execute whatever passed in as msgid and never returns
// (exits if childprocess exits)
// this is the 2nd part of the server (execed by main, this part is running unprivileged)
func Execute() {
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

	// change to my working directory
	err = os.Chdir(srp.WorkingDir)
	if err != nil {
		fmt.Printf("Failed to Chdir() to %s: %s\n", srp.WorkingDir, err)
	}
	fmt.Printf("Chdir() to %s\n", srp.WorkingDir)
	// download the binary and/or archive
	binary := "executable"
	fmt.Printf("Downloading binary from %s\n", srp.URL)
	err = downloadFromURL(srp.URL, binary)
	if err != nil {
		fmt.Printf("Failed to download from %s: %s\n", srp.URL, err)
		os.Exit(10)
	}

	// execute the binary
	ports := countPortCommands(srp.Args)

	resources, err := cl.AllocResources(ctx, &pb.ResourceRequest{Msgid: *msgid, Ports: int32(ports)})
	if err != nil {
		fmt.Printf("Failed to alloc resources: %s\n", err)
		os.Exit(10)
	}
	fmt.Printf("Start commandline: %s %v (%d ports)\n", srp.Binary, srp.Args, ports)
	rArgs := replacePorts(srp.Args, resources.Ports)
	fmt.Printf("Start commandline: %s %v (%d ports)\n", srp.Binary, rArgs, ports)
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
	_, err = cl.Started(ctx, &pb.StartedRequest{Msgid: *msgid})
	if err != nil {
		fmt.Printf("Failed to inform daemon about pending startup. aborting. (%s)\n", err)
		os.Exit(10)
	}
	err = cmd.Run()
	/*
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			fmt.Printf("Failed to get stdout of command %s: %s\n", fullb, err)
			os.Exit(10)
		}
		err = cmd.Start()
		if err != nil {
			fmt.Printf("Failed to start command %s: %s\n", fullb, err)
			os.Exit(10)
		}
		// set up our linereader so we capture stdout and forward to our daemon
		lineOut := new(LineReader)
		buf := make([]byte, 2)
		for {
			ct, err := stdout.Read(buf)
			if err != nil {
				fmt.Printf("Failed to read command output: %s\n", err)
				break
			}
			line := lineOut.gotBytes(buf, ct)
			if line != "" {
				fmt.Printf(">>>>DIRECTCOMMAND: %s: %s\n", fullb, line)
			}
		}

		err = cmd.Wait()
	*/
	fmt.Printf("Command completed: %s\n", err)
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

func downloadFromURL(url string, target string) error {
	fileName := target
	fmt.Println("Downloading", url, "to", fileName)

	// TODO: check file existence first with io.IsExist
	output, err := os.Create(fileName)
	if err != nil {
		fmt.Println("Error while creating", fileName, "-", err)
		return err
	}
	defer output.Close()

	response, err := http.Get(url)
	if err != nil {
		fmt.Println("Error while downloading", url, "-", err)
		return err
	}
	defer response.Body.Close()

	n, err := io.Copy(output, response.Body)
	if err != nil {
		fmt.Println("Error while downloading", url, "-", err)
		return err
	}

	fmt.Println(n, "bytes downloaded.")
	return nil
}
