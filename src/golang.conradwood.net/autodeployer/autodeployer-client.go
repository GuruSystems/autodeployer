package main

// instruct the autodeployer on a given server to download & deploy stuff

import (
	"flag"
	"fmt"
	pb "golang.conradwood.net/autodeployer/proto"
	"golang.conradwood.net/client"
)

// static variables for flag parser
var (
	downloadurl = flag.String("url", "", "The `URL` of the binary to deploy")
	binary      = flag.String("paras", "", "The relative path to the binary to deploy")
	buildid     = flag.Int("build", 1, "The BuildID of the binary to be deployed")
	repo        = flag.String("repo", "", "The name of the repository where the source of the binary to be deployed lives.")
)

func main() {
	flag.Parse()
	conn, err := client.DialWrapper("autodeployer.AutoDeployer")
	if err != nil {
		fmt.Println("failed to dial: %v", err)
		return
	}
	defer conn.Close()
	ctx := client.SetAuthToken()

	cl := pb.NewAutoDeployerClient(conn)
	req := pb.DeployRequest{DownloadURL: *downloadurl,
		Binary:     *binary,
		BuildID:    uint64(*buildid),
		Repository: *repo}
	resp, err := cl.Deploy(ctx, &req)
	if err != nil {
		fmt.Printf("Failed to deploy %s-%d from %s: %s\n", req.Repository, req.BuildID, req.DownloadURL, err)
		return
	}
	fmt.Printf("Response to createvpn: %v\n", resp)
}
