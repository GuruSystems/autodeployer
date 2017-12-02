package main

// instruct the autodeployer on a given server to download & deploy stuff

import (
	"flag"
	"fmt"
	"golang.conradwood.net/client"
	pb "golang.conradwood.net/deploymonkey/proto"
	"google.golang.org/grpc"
	"os"
)

// static variables for flag parser
var (
	filename   = flag.String("configfile", "", "the yaml config file to submit to server")
	namespace  = flag.String("namespace", "", "namespace of the group to update")
	groupname  = flag.String("groupname", "", "groupname of the group to update")
	repository = flag.String("repo", "", "repository of the app in the group to update")
	buildid    = flag.Int("buildid", 0, "the new buildid of the app in the group to update")
)

func main() {
	flag.Parse()

	done := false
	if *filename != "" {
		processFile()
		done = true
	}
	if *namespace != "" {
		updateApp()
	}
	if !done {
		listConfig()
		fmt.Printf("Nothing to do.\n")
	}
	os.Exit(1)
}
func bail(err error, msg string) {
	if err == nil {
		return
	}
	fmt.Printf("%s: %s\n", msg, err)
	os.Exit(10)
}

func listConfig() {
	conn, err := client.DialWrapper("deploymonkey.DeployMonkey")
	if err != nil {
		fmt.Println("failed to dial: %v", err)
		return
	}
	defer conn.Close()
	ctx := client.SetAuthToken()

	cl := pb.NewDeployMonkeyClient(conn)

	ns, err := cl.GetNameSpaces(ctx, &pb.GetNameSpaceRequest{})
	bail(err, "Error getting namespaces")
	fmt.Printf("Namespaces:\n")
	for _, n := range ns.NameSpaces {
		gns, err := cl.GetGroups(ctx, &pb.GetGroupsRequest{NameSpace: n})
		bail(err, "Error getting group")
		fmt.Printf("  %s (%d groups)\n", n, len(gns.GroupNames))
		for _, gs := range gns.GroupNames {
			fmt.Printf("      %s\n", gs)
		}
	}
}

func updateApp() {
	ad := pb.ApplicationDefinition{
		Repository: *repository,
		BuildID:    uint64(*buildid),
	}
	uar := pb.UpdateAppRequest{
		GroupID:   *groupname,
		Namespace: *namespace,
		App:       &ad,
	}
	conn, err := client.DialWrapper("deploymonkey.DeployMonkey")
	if err != nil {
		fmt.Println("failed to dial: %v", err)
		return
	}
	defer conn.Close()
	ctx := client.SetAuthToken()

	cl := pb.NewDeployMonkeyClient(conn)
	resp, err := cl.UpdateApp(ctx, &uar)
	if err != nil {
		fmt.Printf("Failed to update app: %s\n", err)
		return
	}
	fmt.Printf("Response to updateapp: %v\n", resp.Result)
}

func processFile() {
	fd, err := ParseFile(*filename)
	if err != nil {
		fmt.Printf("Failed to parse file %s: %s\n", *filename, err)
		os.Exit(10)
	}

	grpc.EnableTracing = true
	conn, err := client.DialWrapper("deploymonkey.DeployMonkey")
	if err != nil {
		fmt.Println("failed to dial: %v", err)
		return
	}
	defer conn.Close()
	ctx := client.SetAuthToken()

	cl := pb.NewDeployMonkeyClient(conn)
	for _, req := range fd.Groups {
		resp, err := cl.DefineGroup(ctx, req)
		if err != nil {
			fmt.Printf("Failed to define group: %s\n", err)
			return
		}
		if resp.Result != pb.GroupResponseStatus_CHANGEACCEPTED {
			fmt.Printf("Response to deploy: %s - skipping\n", resp.Result)
			continue
		}
		dr := pb.DeployRequest{VersionID: resp.VersionID}
		dresp, err := cl.DeployVersion(ctx, &dr)
		if err != nil {
			fmt.Printf("Failed to deploy version %d: %s\n", resp.VersionID, err)
			return
		}
		fmt.Printf("Deploy response: %v\n", dresp)
	}
}
