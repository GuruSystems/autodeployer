package main

// instruct the autodeployer on a given server to download & deploy stuff

import (
	"flag"
	"fmt"
	pb "golang.conradwood.net/autodeployer/proto"
	"golang.conradwood.net/client"
	"google.golang.org/grpc"
	"strings"
)

// static variables for flag parser
var (
	downloaduser = flag.String("user", "", "the username to authenticate with at the downloadurl")
	downloadpw   = flag.String("password", "", "the password to authenticate with at the downloadurl")
	downloadurl  = flag.String("url", "", "The `URL` of the binary to deploy")
	binary       = flag.String("binary", "", "The relative path to the binary to deploy")
	paras        = flag.String("paras", "", "The parameters to pass to the binary")
	buildid      = flag.Int("build", 1, "The BuildID of the binary to be deployed")
	repo         = flag.String("repo", "", "The name of the repository where the source of the binary to be deployed lives.")
	deployid     = flag.String("deploy_id", "", "an opaque token that is linked to this particular deployment (and returned in deploymentrequest")
)

func main() {
	flag.Parse()
	grpc.EnableTracing = true
	conn, err := client.DialWrapper("autodeployer.AutoDeployer")
	if err != nil {
		fmt.Println("failed to dial: %v", err)
		return
	}
	defer conn.Close()
	ctx := client.SetAuthToken()

	cl := pb.NewAutoDeployerClient(conn)
	req := pb.DeployRequest{DownloadURL: *downloadurl,
		Binary:           *binary,
		BuildID:          uint64(*buildid),
		DownloadUser:     *downloaduser,
		DownloadPassword: *downloadpw,
		Repository:       *repo,
		DeploymentID:     *deployid}
	if *paras != "" {
		args := strings.Split(*paras, " ")
		req.Args = args
	}

	for i, para := range req.Args {
		fmt.Printf("Arg #%d %s\n", i, para)
	}
	resp, err := cl.Deploy(ctx, &req)
	if err != nil {
		fmt.Printf("Failed to deploy %s-%d from %s: %s\n", req.Repository, req.BuildID, req.DownloadURL, err)
		return
	}
	fmt.Printf("Response to deploy: %v\n", resp)
}
