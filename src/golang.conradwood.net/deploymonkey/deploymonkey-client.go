package main

// instruct the autodeployer on a given server to download & deploy stuff

import (
	"flag"
	"fmt"
	"golang.conradwood.net/client"
	pb "golang.conradwood.net/deploymonkey/proto"
	"google.golang.org/grpc"
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
)

func main() {
	flag.Parse()
	grpc.EnableTracing = true
	conn, err := client.DialWrapper("deploymonkey.DeployMonkey")
	if err != nil {
		fmt.Println("failed to dial: %v", err)
		return
	}
	defer conn.Close()
	ctx := client.SetAuthToken()

	cl := pb.NewDeployMonkeyClient(conn)
	req := pb.GroupDefinitionRequest{
		GroupID: "HelloWorldGroup",
	}
	resp, err := cl.DefineGroup(ctx, &req)
	if err != nil {
		fmt.Printf("Failed to define group: %s\n", err)
		return
	}
	fmt.Printf("Response to deploy: %v\n", resp)
}
