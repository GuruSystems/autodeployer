package main

// here we work out what we deployed.
// we essentially return the same diff as a filediff
// so we can use the same "diff" function and apply the diffs to deployment

// we get our state by looking for all registered AutoDeployers in the registry
// we then query each one to figure out what they currently have deployed

import (
	"errors"
	"fmt"
	ad "golang.conradwood.net/autodeployer/proto"
	"golang.conradwood.net/client"
	"golang.org/x/net/context"
	"strings"

	pb "golang.conradwood.net/deploymonkey/proto"
	"google.golang.org/grpc"

	rpb "golang.conradwood.net/registrar/proto"
)

var (
	ctx context.Context
)

func MakeItSo(groupid string, ads []*pb.ApplicationDefinition) error {
	sas, err := getDeployers()
	if err != nil {
		return err
	}
	ctx = client.SetAuthToken()
	deplid := groupid
	fmt.Printf("Looking for services with deplid: %s\n", deplid)
	// this is way... to dumb. we do two steps:
	// 1. shutdown all applications for this group
	// 2. fire up new ones
	// TODO: make it smart and SAFELY apply diffs when and if necessary...

	// stopping stuff...
	for _, sa := range sas {
		fmt.Printf("Querying service at: %s:%d\n", sa.Host, sa.Port)
		conn, err := client.DialService(sa)
		if err != nil {
			fmt.Printf("Failed to connect to service %v", sa)
			return err
		}
		defer conn.Close()
		adc := ad.NewAutoDeployerClient(conn)

		apps, err := getDeployments(adc, sa, deplid)
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
	for _, app := range ads {
		fmt.Printf("Starting %d instances of %s\n", app.Instances, app.Repository)
		instances := 0

		retries := 5
		for uint32(instances) < app.Instances {
			retries--
			if retries == 0 {
				fmt.Printf("Wanted to deploy %d instances of %v, but only deployed %d\n", app.Instances, app, instances)
				break
			}
			for _, sa := range sas {
				err = deployOn(sa, app)
				if err == nil {
					instances++
					break
				}
				fmt.Printf("failed to deploy an instance: %s (retries=%d)\n", err, retries)
			}
		}
	}
	return nil
}

func replaceVars(text string, vars map[string]string) string {
	s := text
	for k, v := range vars {
		s = strings.Replace(s, fmt.Sprintf("${%s}", k), v, -1)
	}
	return s
}

func deployOn(sa *rpb.ServiceAddress, app *pb.ApplicationDefinition) error {
	fmt.Printf("Deploying %v on %v\n", app, sa)
	conn, err := client.DialService(sa)
	if err != nil {
		fmt.Printf("Failed to connect to service %v", sa)
		return err
	}
	defer conn.Close()

	vars := make(map[string]string)
	vars["BUILDID"] = fmt.Sprintf("%d", app.BuildID)
	vars["REPOSITORY"] = app.Repository

	adc := ad.NewAutoDeployerClient(conn)
	dr := ad.DeployRequest{
		DownloadURL:      replaceVars(app.DownloadURL, vars),
		DownloadUser:     app.DownloadUser,
		DownloadPassword: app.DownloadPassword,
		Binary:           app.Binary,
		Args:             app.Args,
		Repository:       app.Repository,
		BuildID:          app.BuildID,
		DeploymentID:     app.DeploymentID,
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
		if dr.DeploymentID != deplid {
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

// get all registered deployers and their RPC address
func getDeployers() ([]*rpb.ServiceAddress, error) {
	opts := []grpc.DialOption{grpc.WithInsecure()}
	conn, err := grpc.Dial(client.GetRegistryAddress(), opts...)
	if err != nil {
		fmt.Printf("Error dialling registry @ %s\n", client.GetRegistryAddress())
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
