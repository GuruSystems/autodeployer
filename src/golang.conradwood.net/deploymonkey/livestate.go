package main

// here we work out what we deployed.
// we essentially return the same diff as a filediff
// so we can use the same "diff" function and apply the diffs to deployment

// we get our state by looking for all registered AutoDeployers in the registry
// we then query each one to figure out what they currently have deployed

// we should keep the dependency on other services to a minimum
// afterall this is where we deploy them, so they might not be available
// ATM we need registry, auth and autodeploy

import (
	"errors"
	"fmt"
	ad "golang.conradwood.net/autodeployer/proto"
	"golang.conradwood.net/client"
	"golang.conradwood.net/cmdline"
	"golang.org/x/net/context"
	"strings"

	pb "golang.conradwood.net/deploymonkey/proto"
	"google.golang.org/grpc"

	rpb "golang.conradwood.net/registrar/proto"
)

const (
	DEPLOY_PREFIX = "DM-APPDEF-"
)

var (
	ctx context.Context
)

// this is the most simplest, but definitely not The Right Thing to do
// how it *should* work:
// * work out what is currently deployed
// * work out a difference
// * fire up all the additional ones required (in parallel)
// * retry failured deployments on different servers
// * if any deployment failed: clear the "new ones" again and abort
// * if all succeeded:
// * clear those which are no longer needed (e.g. old ones in a lower version)
func MakeItSo(group *DBGroup, ads []*pb.ApplicationDefinition) error {
	sas, err := getDeployers()
	if err != nil {
		return err
	}
	ctx = client.SetAuthToken()
	// deploymentid is "PREFIX-GroupID-BuildID"
	// stop all for groupid
	stopPrefix := fmt.Sprintf("%s-%d-", DEPLOY_PREFIX, group.id)
	fmt.Printf("Looking for services to stop with deployment id prefix: \"%s\"\n", stopPrefix)
	// this is way... to dumb. we do two steps:
	// 1. shutdown all applications for this group
	// 2. fire up new ones
	// TODO: make it smart and SAFELY apply diffs when and if necessary...

	// stopping stuff...
	for _, sa := range sas {
		//fmt.Printf("Querying service at: %s:%d\n", sa.Host, sa.Port)
		conn, err := client.DialService(sa)
		if err != nil {
			fmt.Printf("Failed to connect to service %v", sa)
			return err
		}
		defer conn.Close()
		adc := ad.NewAutoDeployerClient(conn)

		apps, err := getDeployments(adc, sa, stopPrefix)
		if err != nil {
			return errors.New(fmt.Sprintf("Unable to get deployments from %v: %s", sa, err))
		}
		for _, appid := range apps {
			fmt.Printf("Shutting down: %v\n", appid)
			ud := ad.UndeployRequest{ID: appid}
			_, err = adc.Undeploy(ctx, &ud)
			if err != nil {
				fmt.Printf("Failed to shutdown %v @ %s:\n", sa, appid, err)
			}
		}
	}

	// starting stuff
	// also, this should start them up multi-threaded... and bla
	err = nil
	workeridx := 0
	for _, app := range ads {
		mgroup := app.Machines
		fsas, err := getDeployersInGroup(mgroup, sas)
		if err != nil {
			fmt.Printf("Could not get deployers for group %s: %s\n", mgroup, err)
		}
		if (fsas == nil) || (len(fsas) == 0) {
			s := fmt.Sprintf("No deployers to deploy on for group %s (app=%v)", mgroup, ads)
			fmt.Println(s)
			return errors.New(s)
		}
		workers := len(fsas)
		fmt.Printf("Got %d hosts to deploy on\n", workers)
		fmt.Printf("Starting %d instances of %s\n", app.Instances, app.Repository)
		instances := 0

		retries := 5
		for uint32(instances) < app.Instances {
			if retries == 0 {
				s := fmt.Sprintf("Wanted to deploy %d instances of %v, but only deployed %d", app.Instances, app, instances)
				fmt.Println(s)
				err = errors.New(s)
				break
			}
			workeridx++
			if workeridx >= workers {
				workeridx = 0
			}
			terr := deployOn(fsas[workeridx], group, app)
			if terr == nil {
				instances++
				retries = 5
				continue
			}
			retries--
			fmt.Printf("failed to deploy an instance: %s (retries=%d)\n", terr, retries)
		}
	}
	return err
}

func replaceVars(text string, vars map[string]string) string {
	s := text
	for k, v := range vars {
		s = strings.Replace(s, fmt.Sprintf("${%s}", k), v, -1)
	}
	return s
}

func deployOn(sa *rpb.ServiceAddress, group *DBGroup, app *pb.ApplicationDefinition) error {
	fmt.Printf("Deploying app on host %s:\n", sa.Host)
	PrintApp(app)
	conn, err := client.DialService(sa)
	if err != nil {
		fmt.Printf("Failed to connect to service %v", sa)
		return err
	}
	defer conn.Close()

	vars := make(map[string]string)
	vars["BUILDID"] = fmt.Sprintf("%d", app.BuildID)
	vars["REPOSITORY"] = app.Repository
	deplid := fmt.Sprintf("%s-%d-%d", DEPLOY_PREFIX, group.id, app.BuildID)

	adc := ad.NewAutoDeployerClient(conn)
	dr := ad.DeployRequest{
		DownloadURL:      replaceVars(app.DownloadURL, vars),
		DownloadUser:     app.DownloadUser,
		DownloadPassword: app.DownloadPassword,
		Binary:           app.Binary,
		Args:             app.Args,
		Repository:       app.Repository,
		BuildID:          app.BuildID,
		DeploymentID:     deplid,
		Namespace:        group.groupDef.Namespace,
		Groupname:        group.groupDef.GroupID,
		AutoRegistration: app.AutoRegs,
	}
	dres, err := adc.Deploy(ctx, &dr)
	if err != nil {
		fmt.Printf("failed to deploy %v on %v: %s\n", app, adc, err)
		return err
	}
	if !dres.Success {
		s := fmt.Sprintf("failed to startup app %v\n", app)
		fmt.Println(s)
		return errors.New(s)
	}
	fmt.Printf("Successfully deployed %v on %s", app, sa.Host)
	return nil

}

func getDeployments(adc ad.AutoDeployerClient, sa *rpb.ServiceAddress, deplid string) ([]string, error) {
	//	var res []*pb.ApplicationDefinition
	var res []string
	info, err := adc.GetDeployments(ctx, &ad.InfoRequest{})
	if err != nil {
		fmt.Printf("Failed to query service %v", sa)
		return nil, err
	}
	for _, app := range info.Apps {
		dr := app.Deployment
		if !strings.HasPrefix(dr.DeploymentID, deplid) {
			continue
		}
		res = append(res, app.ID)
		/*
			ad := pb.ApplicationDefinition{
				DownloadURL:      dr.DownloadURL,
				DownloadUser:     dr.DownloadUser,
				DownloadPassword: dr.DownloadPassword,
				Binary:           dr.Binary,
				Args:             dr.Args,
				Repository:       dr.Repository,
				BuildID:          dr.BuildID,
				DeploymentID:     dr.DeploymentID,
			}

			res = append(res, &ad)
			fmt.Printf("Deployment: %v\n", dr)
		*/
	}
	return res, nil
}

// given a name will only return deployers in that group name
// if name == "" it will be assumed to be "worker"
func getDeployersInGroup(name string, all []*rpb.ServiceAddress) ([]*rpb.ServiceAddress, error) {
	var res []*rpb.ServiceAddress

	if name == "" {
		name = "worker"
	}
	for _, sa := range all {
		conn, err := client.DialService(sa)
		if err != nil {
			fmt.Printf("Failed to connect to service %v", sa)
			continue
		}
		adc := ad.NewAutoDeployerClient(conn)
		req := &ad.MachineInfoRequest{}
		mir, err := adc.GetMachineInfo(ctx, req)
		if err != nil {
			conn.Close()
			fmt.Printf("Failed to get machine info on %v\n", sa)
			continue
		}
		conn.Close()
		mg := mir.MachineGroup
		if mg == "" {
			mg = "worker"
		}
		fmt.Printf("Autodeployer on %s is in group %s (requested: %s)\n", sa.Host, mg, name)

		if mg == name {
			res = append(res, sa)
		}
	}

	return res, nil
}

// get all registered deployers and their RPC address
func getDeployers() ([]*rpb.ServiceAddress, error) {
	opts := []grpc.DialOption{grpc.WithInsecure()}
	conn, err := grpc.Dial(cmdline.GetRegistryAddress(), opts...)
	if err != nil {
		fmt.Printf("Error dialling registry @ %s\n", cmdline.GetRegistryAddress())
		return nil, err
	}
	defer conn.Close()
	rcl := rpb.NewRegistryClient(conn)
	ctx := client.SetAuthToken()
	lr := rpb.ListRequest{}
	lr.Name = "autodeployer.AutoDeployer"
	resp, err := rcl.ListServices(ctx, &lr)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Error getting services: %s", err))
	}
	var res []*rpb.ServiceAddress
	for _, sd := range resp.Service {
		loc := sd.Location
		for _, sa := range loc.Address {
			res = append(res, sa)
		}
	}
	return res, nil
}
