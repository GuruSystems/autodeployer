package main

// instruct the autodeployer on a given server to download & deploy stuff

import (
	"errors"
	"flag"
	"fmt"
	"golang.conradwood.net/client"
	pb "golang.conradwood.net/deploymonkey/proto"
	"google.golang.org/grpc"
	"os"
)

// static variables for flag parser
var (
	filename      = flag.String("configfile", "", "the yaml config file to submit to server")
	namespace     = flag.String("namespace", "", "namespace of the group to update")
	groupname     = flag.String("groupname", "", "groupname of the group to update")
	repository    = flag.String("repository", "", "repository of the app in the group to update")
	buildid       = flag.Int("buildid", 0, "the new buildid of the app in the group to update")
	binary        = flag.String("binary", "", "the binary of the app in the group to update")
	apply_version = flag.Int("apply_version", 0, "(re-)apply a given version")
	applyall      = flag.Bool("apply_all", false, "reapply ALL groups")
	applypending  = flag.Bool("apply_pending", false, "reapply any pending group versions")
)

func main() {
	flag.Parse()

	done := false
	if *applyall || *applypending {
		applyVersions()
		done = true
	}
	if *apply_version != 0 {
		applyVersion()
		done = true
	}
	if *filename != "" {
		processFile()
		done = true
	}
	if *namespace != "" {
		if *binary != "" {
			updateApp()
		} else {
			updateRepo()
		}
		done = true
	}
	if !done {
		listConfig()
		fmt.Printf("Nothing to do.\n")
		os.Exit(1)
	}
	os.Exit(0)
}
func bail(err error, msg string) {
	if err == nil {
		return
	}
	fmt.Printf("%s: %s\n", msg, err)
	os.Exit(10)
}

func applyVersions() {
	conn, err := client.DialWrapper("deploymonkey.DeployMonkey")
	if err != nil {
		fmt.Println("failed to dial: %v", err)
		return
	}
	defer conn.Close()
	ctx := client.SetAuthToken()

	cl := pb.NewDeployMonkeyClient(conn)
	all := *applyall
	dr := pb.ApplyRequest{All: all}
	resp, err := cl.ApplyVersions(ctx, &dr)
	if err != nil {
		fmt.Printf("Error: %s\n", err)
		return
	}
	fmt.Printf("Response: %v\n", resp)
}

func applyVersion() {
	conn, err := client.DialWrapper("deploymonkey.DeployMonkey")
	if err != nil {
		fmt.Println("failed to dial: %v", err)
		return
	}
	defer conn.Close()
	ctx := client.SetAuthToken()

	cl := pb.NewDeployMonkeyClient(conn)
	dr := pb.DeployRequest{VersionID: fmt.Sprintf("%d", *apply_version)}
	resp, err := cl.DeployVersion(ctx, &dr)
	if err != nil {
		fmt.Printf("Error: %s\n", err)
		return
	}
	fmt.Printf("Response: %v\n", resp)
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
		fmt.Printf("  %s (%d groups)\n", n, len(gns.Groups))
		for _, gs := range gns.Groups {
			gar := pb.GetAppsRequest{
				NameSpace: gs.NameSpace,
				GroupName: gs.GroupID}
			gapps, err := cl.GetApplications(ctx, &gar)
			bail(err, "Failed to get applications")
			marker := ""
			if gs.PendingVersion != gs.DeployedVersion {
				marker = " ** <-- **"
			}
			fmt.Printf("      %s (%d applications)%s\n", gs, len(gapps.Applications), marker)
			for _, app := range gapps.Applications {
				fmt.Printf("           %dx Repo=%s, Binary=%s, BuildID=#%d, %d autoregistrations\n", app.Instances, app.Repository, app.Binary, app.BuildID, len(app.AutoRegs))
			}
		}
	}
}

func updateRepo() {
	if *namespace == "" {
		bail(errors.New("Namespace required"), "Cannot update repo")
	}
	if *groupname == "" {
		bail(errors.New("Groupname required"), "Cannot update repo")
	}
	if *repository == "" {
		bail(errors.New("Repository required"), "Cannot update repo")
	}
	if *buildid == 0 {
		bail(errors.New("BuildID required"), "Cannot update repo")
	}
	fmt.Printf("Updating all apps in repo %s in group %s to buildid %d\n", *repository, *groupname, *buildid)
	ur := pb.UpdateRepoRequest{
		Namespace:  *namespace,
		GroupID:    *groupname,
		Repository: *repository,
		BuildID:    uint64(*buildid),
	}
	conn, err := client.DialWrapper("deploymonkey.DeployMonkey")
	bail(err, "Failed to dial")
	defer conn.Close()
	ctx := client.SetAuthToken()
	cl := pb.NewDeployMonkeyClient(conn)
	resp, err := cl.UpdateRepo(ctx, &ur)
	bail(err, "Failed to update repo")
	fmt.Printf("Response to updaterepo: %v\n", resp)
	return
}

func updateApp() {
	ad := pb.ApplicationDefinition{
		Repository: *repository,
		Binary:     *binary,
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
